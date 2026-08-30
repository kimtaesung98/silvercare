import { Injectable, Logger } from '@nestjs/common';
import { PrismaService } from '../../prisma/prisma.service';
import { EmotionTone, KeywordCategory } from '@prisma/client';

interface ExtractedKeyword {
  keyword: string;
  category: KeywordCategory;
  emotionTone: EmotionTone;
}

/**
 * 관심사 키워드 엔진.
 *
 * 설계 원칙 (기획 논의 반영):
 * - 사전 등록형 고정 프로필이 아니라, 대화에서 빈도 높은 키워드를 태그로 동적 업데이트한다.
 * - 시간 감쇠(time-decay) 가중치: 오래된 언급보다 최근 언급에 더 높은 점수를 부여한다.
 *   score_new = score_old * DECAY_FACTOR + mention_count_today
 * - 다음 화제 선택 시 "최근 N일 내 반복 사용하지 않은 태그" 우선으로 로테이션한다.
 */
@Injectable()
export class KeywordService {
  private readonly logger = new Logger(KeywordService.name);
  private readonly DECAY_FACTOR = 0.9;
  private readonly RECENT_ROTATION_DAYS = 3;

  constructor(private readonly prisma: PrismaService) {}

  /**
   * 세션 종료 후 배치로 호출.
   * 1차: 형태소 분석기(Kiwi 등, 별도 Python 서브서비스)로 명사 후보 추출 (여기서는 인터페이스만 정의)
   * 2차: LLM에게 문맥 기반으로 "대화 주제가 될 만한 키워드"를 정제 요청 (session 모듈과 별도 호출)
   */
  async extractAndUpdateFromSession(sessionId: string, elderId: string) {
    const utterances = await this.prisma.utterance.findMany({
      where: { sessionId, speaker: 'ELDER' },
      orderBy: { createdAt: 'asc' },
    });

    if (utterances.length === 0) {
      this.logger.log(`sessionId=${sessionId}: 노인 발화 없음, 키워드 갱신 스킵`);
      return;
    }

    // TODO: 실제로는 Kiwi 형태소 분석 + Claude Haiku 2차 정제 파이프라인으로 대체
    const extracted = await this.extractKeywordsStub(utterances.map((u) => u.text));

    for (const item of extracted) {
      await this.upsertKeywordTag(elderId, item);
    }
  }

  private async upsertKeywordTag(elderId: string, item: ExtractedKeyword) {
    const existing = await this.prisma.keywordTag.findUnique({
      where: { elderId_keyword: { elderId, keyword: item.keyword } },
    });

    if (!existing) {
      await this.prisma.keywordTag.create({
        data: {
          elderId,
          keyword: item.keyword,
          category: item.category,
          score: 1,
          emotionTone: item.emotionTone,
          lastMentionedAt: new Date(),
          mentionCountTotal: 1,
        },
      });
      return;
    }

    const decayedScore = existing.score * this.DECAY_FACTOR + 1;

    await this.prisma.keywordTag.update({
      where: { id: existing.id },
      data: {
        score: decayedScore,
        emotionTone: item.emotionTone, // 가장 최근 반응 톤으로 갱신
        lastMentionedAt: new Date(),
        mentionCountTotal: existing.mentionCountTotal + 1,
      },
    });
  }

  /**
   * 다음 대화 화제 후보 조회.
   * 점수 상위 태그 중, 최근 RECENT_ROTATION_DAYS일 내 사용하지 않은 것을 우선한다.
   */
  async getNextTopicCandidates(elderId: string, limit = 5) {
    const cutoff = new Date(Date.now() - this.RECENT_ROTATION_DAYS * 24 * 60 * 60 * 1000);

    const rotated = await this.prisma.keywordTag.findMany({
      where: { elderId, lastMentionedAt: { lt: cutoff }, emotionTone: { not: EmotionTone.AVOIDANT } },
      orderBy: { score: 'desc' },
      take: limit,
    });

    if (rotated.length > 0) return rotated;

    // 로테이션 대상이 없으면 회피 태그를 제외한 상위 점수 태그로 폴백
    return this.prisma.keywordTag.findMany({
      where: { elderId, emotionTone: { not: EmotionTone.AVOIDANT } },
      orderBy: { score: 'desc' },
      take: limit,
    });
  }

  private async extractKeywordsStub(_texts: string[]): Promise<ExtractedKeyword[]> {
    // 실제 구현 전 임시 목업
    return [];
  }
}
