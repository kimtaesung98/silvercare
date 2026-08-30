import { Injectable, Logger } from '@nestjs/common';
import Anthropic from '@anthropic-ai/sdk';
import { PrismaService } from '../../prisma/prisma.service';
import { Speaker } from '@prisma/client';

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

  constructor(private readonly prisma: PrismaService) {
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
      { role: 'user', content: '대화를 시작해줘. 노인에게 먼저 친근하게 인사하고 관심사 화제를 자연스럽게 꺼내줘.' },
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
    messages: { role: 'user' | 'assistant'; content: string }[],
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
