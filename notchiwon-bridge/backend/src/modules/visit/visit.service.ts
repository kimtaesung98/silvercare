import { Injectable, Logger, NotFoundException } from '@nestjs/common';
import { PrismaService } from '../../prisma/prisma.service';
import { VisitStatus } from '@prisma/client';

interface LocationUpdateInput {
  caregiverId: string;
  visitId: string;
  latitude: number;
  longitude: number;
}

/**
 * ETA 엔진.
 * 실제 구현 시 카카오모빌리티/티맵 API로 도로 상황 기반 ETA를 계산하고,
 * 임계값(SESSION_TRIGGER_ETA_MINUTES) 이내 진입 시 세션 오케스트레이터에
 * 이벤트를 발행한다 (Redis Pub/Sub 또는 EventEmitter 사용 예정).
 */
@Injectable()
export class VisitService {
  private readonly logger = new Logger(VisitService.name);

  constructor(private readonly prisma: PrismaService) {}

  async handleLocationUpdate(input: LocationUpdateInput) {
    // TODO: 외부 지도 API(카카오모빌리티/티맵)로 실제 ETA 계산
    const etaMinutes = await this.calculateEtaMinutesStub(input);

    // 상태를 갱신하기 전에 기존 상태를 읽어야 "이미 세션 진행중" 여부를 판단할 수 있다.
    const current = await this.prisma.visit.findUnique({
      where: { id: input.visitId },
      select: { status: true },
    });
    if (!current) {
      throw new NotFoundException(`visit ${input.visitId} not found`);
    }

    // 예정/이동중일 때만 EN_ROUTE로 전환한다. 세션진행중·완료·취소 상태는 덮어쓰지 않는다.
    const canTrigger =
      current.status === VisitStatus.SCHEDULED || current.status === VisitStatus.EN_ROUTE;

    const visit = await this.prisma.visit.update({
      where: { id: input.visitId },
      data: {
        etaCurrent: new Date(Date.now() + etaMinutes * 60 * 1000),
        ...(canTrigger && { status: VisitStatus.EN_ROUTE }),
      },
    });

    const thresholdMinutes = Number(process.env.SESSION_TRIGGER_ETA_MINUTES ?? 15);

    if (etaMinutes <= thresholdMinutes && canTrigger) {
      this.logger.log(
        `ETA ${etaMinutes}분 <= 임계값 ${thresholdMinutes}분 → 세션 개시 이벤트 발행 (visitId=${visit.id})`,
      );
      // TODO: SessionService.startSession(visit.id) 호출 또는 이벤트 버스로 발행
    }

    return { visitId: visit.id, etaMinutes };
  }

  private async calculateEtaMinutesStub(_input: LocationUpdateInput): Promise<number> {
    // 실제 구현 전까지의 임시 목업. 실제로는 지도 API 응답의 duration 사용.
    return 12;
  }
}
