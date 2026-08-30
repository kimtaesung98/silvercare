import { Module } from '@nestjs/common';
import { BriefingService } from './briefing.service';

@Module({
  providers: [BriefingService],
  exports: [BriefingService],
})
export class BriefingModule {}
