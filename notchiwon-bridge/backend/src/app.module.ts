import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { PrismaModule } from './prisma/prisma.module';
import { VisitModule } from './modules/visit/visit.module';
import { SessionModule } from './modules/session/session.module';
import { KeywordModule } from './modules/keyword/keyword.module';
import { BriefingModule } from './modules/briefing/briefing.module';
import { EscalationModule } from './modules/escalation/escalation.module';

@Module({
  imports: [
    ConfigModule.forRoot({ isGlobal: true }),
    PrismaModule,
    VisitModule,
    SessionModule,
    KeywordModule,
    BriefingModule,
    EscalationModule,
  ],
})
export class AppModule {}
