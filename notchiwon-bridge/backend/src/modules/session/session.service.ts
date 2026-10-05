import { BadRequestException, Injectable, Logger, NotFoundException } from '@nestjs/common';
import Anthropic from '@anthropic-ai/sdk';
import { PrismaService } from '../../prisma/prisma.service';
import { EscalationService } from '../escalation/escalation.service';
import { EscalationTriggerType, Speaker } from '@prisma/client';

type ChatMessage = { role: 'user' | 'assistant'; content: string };

const SESSION_KICKOFF_PROMPT = '대화를 시작해줘. 노인에게 먼저 친근하게 인사하고 관심사 화제를 자연스럽게 꺼내줘.';

// 위급 발화 감지 시 LLM을 거치지 않고 즉시 내보내는 고정 안심 응답
const ESCALATION_REPLY = '지금 많이 힘드시죠. 선생님께 바로 알려드렸어요. 조금만 기다려주세요.';

// LLM 호출 실패·빈 응답 시 대화가 끊기지 않도록 쓰는 폴백 응답
const FALLBACK_REPLY = '네, 말씀 잘 들었어요. 선생님이 곧 오실 거예요. 조금만 더 저랑 이야기 나눠요.';

export interface ElderUtteranceResult {
  elderUtteranceId: string;
  aiUtteranceId: string;
  reply: string;
  escalated: boolean;
  triggerType: EscalationTriggerType | null;
}

/**
 * 대화 오케스트레이터.
 * - 세션 시작 시 노인의 상위 관심사 키워드를 조회해 시스템 프롬프트에 주입
 * - 노인 발화(STT 결과)를 받아 안전 필터 통과 후 Claude 응답 생성
 * - 모든 발화를 utterance 테이블에 로깅
 *
 * 안전장치: 위급 신호(낙상/통증/자타해 관련 발화)는 이 서비스 이전 단계인
 * escalation 모듈의 규칙 기반 필터를 반드시 함께 거치도록 구성한다 (LLM 단독 판단 금지).
 */
@Injectable()
export class SessionService {
  private readonly logger = new Logger(SessionService.name);
  private readonly anthropic: Anthropic;

  constructor(
    private readonly prisma: PrismaService,
    private readonly escalationService: EscalationService,
  ) {
    this.anthropic = new Anthropic({ apiKey: process.env.ANTHROPIC_API_KEY });
  }

  async startSession(visitId: string, triggerEtaMinutes: number) {
    const visit = await this.prisma.visit.findUniqueOrThrow({
      where: { id: visitId },
      include: { elder: true },
    });

    const session = await this.prisma.conversationSession.create({
      data: {
        visitId: visit.id,
        elderId: visit.elderId,
        triggerEtaMinutes,
      },
    });

    const topKeywords = await this.prisma.keywordTag.findMany({
      where: { elderId: visit.elderId },
      orderBy: { score: 'desc' },
      take: 5,
    });

    const systemPrompt = this.buildSystemPrompt(visit.elder.name, topKeywords.map((k) => k.keyword));

    const openingMessage = await this.generateAiUtterance(systemPrompt, [
      { role: 'user', content: SESSION_KICKOFF_PROMPT },
    ]);

    await this.prisma.utterance.create({
      data: {
        sessionId: session.id,
        speaker: Speaker.AI,
        text: openingMessage,
      },
    });

    this.logger.log(`세션 시작: sessionId=${session.id}, elder=${visit.elder.name}`);

    return { sessionId: session.id, openingMessage };
  }

  async endSession(sessionId: string, reason: 'CAREGIVER_ARRIVED' | 'ELDER_DECLINED' | 'TIMEOUT' | 'ERROR') {
    await this.prisma.conversationSession.update({
      where: { id: sessionId },
      data: { endedAt: new Date(), endedReason: reason },
    });

    // TODO: keyword 모듈에 세션 종료 이벤트 발행 → 비동기 배치로 키워드 가중치 갱신
    // TODO: briefing 모듈에 세션 종료 이벤트 발행 → 브리핑 리포트 생성
  }

  /**
   * 노인 발화(STT 결과) 1건을 처리한다.
   * 1. 발화를 저장하기 전에 규칙 기반 필터를 먼저 돌린다 (LLM 결과와 무관하게 항상 실행).
   * 2. 위급 신호가 감지되면 에스컬레이션 이벤트를 남기고, LLM 호출 없이 고정 안심 응답을 돌려준다.
   * 3. 그 외에는 지금까지의 대화 기록으로 Claude 응답을 생성한다. 실패 시 폴백 응답을 쓴다.
   */
  async handleElderUtterance(sessionId: string, text: string, audioRef?: string): Promise<ElderUtteranceResult> {
    const trimmed = text?.trim();
    if (!trimmed) {
      throw new BadRequestException('text is required');
    }

    const session = await this.prisma.conversationSession.findUnique({
      where: { id: sessionId },
      include: { elder: true },
    });
    if (!session) {
      throw new NotFoundException(`session ${sessionId} not found`);
    }
    if (session.endedAt) {
      throw new BadRequestException(`session ${sessionId} already ended`);
    }

    const triggerType = this.escalationService.detect(trimmed);

    const elderUtterance = await this.prisma.utterance.create({
      data: {
        sessionId,
        speaker: Speaker.ELDER,
        text: trimmed,
        audioRef,
        flaggedRisk: triggerType !== null,
      },
    });

    let reply: string;
    if (triggerType) {
      await this.escalationService.raiseEscalation(sessionId, elderUtterance.id, triggerType);
      reply = ESCALATION_REPLY;
    } else {
      reply = await this.generateReply(session.elderId, session.elder.name, sessionId);
    }

    const aiUtterance = await this.prisma.utterance.create({
      data: { sessionId, speaker: Speaker.AI, text: reply },
    });

    return {
      elderUtteranceId: elderUtterance.id,
      aiUtteranceId: aiUtterance.id,
      reply,
      escalated: triggerType !== null,
      triggerType,
    };
  }

  private async generateReply(elderId: string, elderName: string, sessionId: string): Promise<string> {
    const [topKeywords, history] = await Promise.all([
      this.prisma.keywordTag.findMany({
        where: { elderId },
        orderBy: { score: 'desc' },
        take: 5,
      }),
      this.prisma.utterance.findMany({
        where: { sessionId },
        orderBy: { createdAt: 'asc' },
      }),
    ]);

    const systemPrompt = this.buildSystemPrompt(elderName, topKeywords.map((k) => k.keyword));
    const messages = this.buildMessages(history);

    try {
      const reply = (await this.generateAiUtterance(systemPrompt, messages)).trim();
      return reply || FALLBACK_REPLY;
    } catch (err) {
      this.logger.error(`Claude 응답 생성 실패: sessionId=${sessionId}`, err instanceof Error ? err.stack : err);
      return FALLBACK_REPLY;
    }
  }

  /**
   * 저장된 발화 기록을 Messages API 형식으로 변환한다.
   * 세션 첫 발화는 AI 인사말이므로 시작 지시문을 user 턴으로 앞에 붙이고,
   * 같은 화자가 연속된 경우(예: 노인이 연달아 말함)는 한 턴으로 합친다.
   */
  private buildMessages(history: { speaker: Speaker; text: string }[]): ChatMessage[] {
    const messages: ChatMessage[] = [{ role: 'user', content: SESSION_KICKOFF_PROMPT }];
    for (const u of history) {
      const role = u.speaker === Speaker.ELDER ? 'user' : 'assistant';
      const last = messages[messages.length - 1];
      if (last.role === role) {
        last.content = `${last.content}\n${u.text}`;
      } else {
        messages.push({ role, content: u.text });
      }
    }
    return messages;
  }

  private buildSystemPrompt(elderName: string, interestKeywords: string[]): string {
    return [
      `당신은 치매를 겪고 계신 ${elderName}님과 대화하는 다정한 AI 동반자입니다.`,
      `역할: 조무사 선생님이 도착하기 전까지 자연스러운 대화로 안정감을 드리는 것입니다.`,
      interestKeywords.length > 0
        ? `${elderName}님이 최근 자주 이야기하신 화제: ${interestKeywords.join(', ')}. 이 중 하나로 자연스럽게 대화를 시작하되, 강요하지 마세요.`
        : `아직 등록된 관심사가 없으니 날씨, 오늘 하루 등 편안한 일상 화제로 시작하세요.`,
      `절대 하지 말 것: 의료적 진단·조언, 복잡한 질문 연속 제시, 부정확한 사실 단정.`,
      `발화가 낙상, 통증, 자해/타해와 관련된 경우 절대 스스로 판단하지 말고 "지금 많이 힘드시죠, 조금만 기다려주세요"처럼 안심시키는 짧은 응답만 하세요 (별도 안전 시스템이 담당자에게 알립니다).`,
      `문장은 짧고 천천히, 한 번에 하나의 질문만 하세요.`,
    ].join('\n');
  }

  private async generateAiUtterance(
    systemPrompt: string,
    messages: ChatMessage[],
  ): Promise<string> {
    const response = await this.anthropic.messages.create({
      model: process.env.CLAUDE_CONVERSATION_MODEL ?? 'claude-sonnet-4-6',
      max_tokens: 300,
      system: systemPrompt,
      messages,
    });

    const textBlock = response.content.find((block) => block.type === 'text');
    return textBlock && 'text' in textBlock ? textBlock.text : '';
  }
}
