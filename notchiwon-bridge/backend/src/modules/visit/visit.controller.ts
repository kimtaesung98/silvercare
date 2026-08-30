import { Body, Controller, Post } from '@nestjs/common';
import { VisitService } from './visit.service';

class LocationUpdateDto {
  caregiverId: string;
  visitId: string;
  latitude: number;
  longitude: number;
}

@Controller('visits')
export class VisitController {
  constructor(private readonly visitService: VisitService) {}

  // 조무사 앱이 5~10초 주기로 호출 (또는 WebSocket으로 대체 예정)
  @Post('location')
  async updateLocation(@Body() dto: LocationUpdateDto) {
    return this.visitService.handleLocationUpdate(dto);
  }
}
