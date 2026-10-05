import { Injectable, Logger } from '@nestjs/common';
import { PrismaService } from '../../prisma/prisma.service';
import { EscalationTriggerType } from '@prisma/client';

/**
 * 위급 상황 감지 — 이중 안전장치의 1차 방어선 (규칙 기반 키워드 필터).
 * LLM 판단에만 의존하지 않고, 명확한 위험 키워드가 발견되면 즉시 에스컬레이션한다.
 *
 * NOTE: 아래 키워드 목록은 초안이며, 실제 서비스 적용 전 의료/요양 도메인 전문가의
 * 검토를 거쳐야 한다 (과소·과대 탐지 균형 조정 필요).
 */
@Injectable()
export class EscalationService {
  private readonly logger = new Logger(EscalationService.name);

  private readonly ruleMap: { pattern: RegExp; type: EscalationTriggerType }[] = [
    { pattern: /(넘어졌|쓰러졌|낙상)/, type: EscalationTriggerType.FALL_MENTION },
    { pattern: /(아파(?!트)|아프|통증|숨이\s*차)/, type: EscalationTriggerType.PAIN_COMPLAINT },
    { pattern: /(죽고\s*싶|자해|때리)/, type: EscalationTriggerType.SELF_OR_OTHER_HARM },
  ];

  constructor(private readonly prisma: PrismaService) {}

  detect(text: string): EscalationTriggerType | null {
    for (const rule of this.ruleMap) {
      if (rule.pattern.test(text)) {
        return rule.type;
      }
    }
    return null;
  }

  async raiseEscalation(
    sessionId: string,
    utteranceId: string,
    triggerType: EscalationTriggerType,
  ) {
    const event = await this.prisma.escalationEvent.create({
      data: {
        sessionId,
        utteranceId,
        triggerType,
        notifiedTargets: { caregiver: true, guardian: false, center: true }, // TODO: 정책에 맞게 조정
      },
    });

    this.logger.warn(`위급 이벤트 발생: sessionId=${sessionId}, type=${triggerType}`);

    // TODO: FCM 푸시로 조무사/센터에 즉시 알림 발송

    return event;
  }
}
