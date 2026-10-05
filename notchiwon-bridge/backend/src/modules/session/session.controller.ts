import { Body, Controller, Param, Post } from '@nestjs/common';
import { SessionService } from './session.service';

@Controller('sessions')
export class SessionController {
  constructor(private readonly sessionService: SessionService) {}

  @Post('start')
  async start(@Body() dto: { visitId: string; triggerEtaMinutes: number }) {
    return this.sessionService.startSession(dto.visitId, dto.triggerEtaMinutes);
  }

  @Post('end')
  async end(@Body() dto: { sessionId: string; reason: 'CAREGIVER_ARRIVED' | 'ELDER_DECLINED' | 'TIMEOUT' | 'ERROR' }) {
    return this.sessionService.endSession(dto.sessionId, dto.reason);
  }

  // 노인용 태블릿이 STT 결과를 발화 단위로 전송
  @Post(':sessionId/utterances')
  async utter(@Param('sessionId') sessionId: string, @Body() dto: { text: string; audioRef?: string }) {
    return this.sessionService.handleElderUtterance(sessionId, dto.text, dto.audioRef);
  }
}
