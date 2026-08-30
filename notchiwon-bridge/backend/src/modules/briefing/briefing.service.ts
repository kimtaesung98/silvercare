import { Injectable } from '@nestjs/common';
import { PrismaService } from '../../prisma/prisma.service';

/**
 * 브리핑 요약 엔진.
 * 세션 종료 직후 또는 조무사 도착 임박 시점에 3~5줄 요약 카드를 생성한다.
 * 경량 모델(Claude Haiku급)을 사용해 응답 속도와 비용을 최적화한다.
 */
@Injectable()
export class BriefingService {
  constructor(private readonly prisma: PrismaService) {}

  async generateBriefing(sessionId: string) {
    const utterances = await this.prisma.utterance.findMany({
      where: { sessionId },
      orderBy: { createdAt: 'asc' },
    });

    // TODO: Claude Haiku 호출로 summaryText, topKeywords, emotionFlag 생성
    const summaryText = this.buildStubSummary(utterances.length);

    return this.prisma.briefingReport.create({
      data: {
        sessionId,
        summaryText,
        topKeywords: [],
        emotionFlag: null,
      },
    });
  }

  private buildStubSummary(utteranceCount: number): string {
    return `총 ${utteranceCount}개 발화가 기록되었습니다. (요약 로직 구현 예정)`;
  }
}
