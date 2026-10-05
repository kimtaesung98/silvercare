import { BadRequestException, NotFoundException } from '@nestjs/common';
import { EscalationTriggerType, Speaker } from '@prisma/client';
import { SessionService } from './session.service';
import { EscalationService } from '../escalation/escalation.service';
import { PrismaService } from '../../prisma/prisma.service';

const mockCreate = jest.fn();
jest.mock('@anthropic-ai/sdk', () => ({
  __esModule: true,
  default: jest.fn().mockImplementation(() => ({ messages: { create: mockCreate } })),
}));

describe('SessionService.handleElderUtterance', () => {
  let prisma: {
    conversationSession: { findUnique: jest.Mock };
    utterance: { create: jest.Mock; findMany: jest.Mock };
    keywordTag: { findMany: jest.Mock };
  };
  let escalation: { detect: jest.Mock; raiseEscalation: jest.Mock };
  let service: SessionService;
  let utteranceSeq: number;

  const activeSession = { id: 's1', elderId: 'e1', endedAt: null, elder: { name: '김순자' } };

  beforeEach(() => {
    jest.clearAllMocks();
    utteranceSeq = 0;
    prisma = {
      conversationSession: { findUnique: jest.fn().mockResolvedValue(activeSession) },
      utterance: {
        create: jest.fn().mockImplementation(({ data }) => Promise.resolve({ id: `u${++utteranceSeq}`, ...data })),
        findMany: jest.fn().mockResolvedValue([]),
      },
      keywordTag: { findMany: jest.fn().mockResolvedValue([{ keyword: '손주' }]) },
    };
    escalation = {
      detect: jest.fn().mockReturnValue(null),
      raiseEscalation: jest.fn().mockResolvedValue({ id: 'ev1' }),
    };
    service = new SessionService(
      prisma as unknown as PrismaService,
      escalation as unknown as EscalationService,
    );
    mockCreate.mockResolvedValue({ content: [{ type: 'text', text: '손주 이야기 더 들려주세요.' }] });
  });

  it('stores the elder utterance, asks Claude with the conversation so far, and stores the reply', async () => {
    prisma.utterance.findMany.mockResolvedValue([
      { speaker: Speaker.AI, text: '안녕하세요, 김순자님!' },
      { speaker: Speaker.ELDER, text: '손주가 어제 왔어' },
    ]);

    const result = await service.handleElderUtterance('s1', '  손주가 어제 왔어 ');

    expect(escalation.detect).toHaveBeenCalledWith('손주가 어제 왔어');
    expect(prisma.utterance.create).toHaveBeenNthCalledWith(1, {
      data: expect.objectContaining({ sessionId: 's1', speaker: Speaker.ELDER, text: '손주가 어제 왔어', flaggedRisk: false }),
    });

    const request = mockCreate.mock.calls[0][0];
    expect(request.system).toContain('김순자');
    expect(request.system).toContain('손주');
    expect(request.messages.map((m: { role: string }) => m.role)).toEqual(['user', 'assistant', 'user']);
    expect(request.messages[2].content).toBe('손주가 어제 왔어');

    expect(prisma.utterance.create).toHaveBeenNthCalledWith(2, {
      data: { sessionId: 's1', speaker: Speaker.AI, text: '손주 이야기 더 들려주세요.' },
    });
    expect(escalation.raiseEscalation).not.toHaveBeenCalled();
    expect(result).toEqual({
      elderUtteranceId: 'u1',
      aiUtteranceId: 'u2',
      reply: '손주 이야기 더 들려주세요.',
      escalated: false,
      triggerType: null,
    });
  });

  it('escalates a risky utterance and replies with a fixed calming line without calling Claude', async () => {
    escalation.detect.mockReturnValue(EscalationTriggerType.FALL_MENTION);

    const result = await service.handleElderUtterance('s1', '화장실에서 넘어졌어');

    expect(prisma.utterance.create).toHaveBeenNthCalledWith(1, {
      data: expect.objectContaining({ speaker: Speaker.ELDER, flaggedRisk: true }),
    });
    expect(escalation.raiseEscalation).toHaveBeenCalledWith('s1', 'u1', EscalationTriggerType.FALL_MENTION);
    expect(mockCreate).not.toHaveBeenCalled();
    expect(result.escalated).toBe(true);
    expect(result.triggerType).toBe(EscalationTriggerType.FALL_MENTION);
    expect(result.reply).toContain('조금만 기다려주세요');
  });

  it('merges consecutive elder utterances into one user turn', async () => {
    prisma.utterance.findMany.mockResolvedValue([
      { speaker: Speaker.AI, text: '안녕하세요!' },
      { speaker: Speaker.ELDER, text: '응' },
      { speaker: Speaker.ELDER, text: '오늘 날씨 좋네' },
    ]);

    await service.handleElderUtterance('s1', '오늘 날씨 좋네');

    const { messages } = mockCreate.mock.calls[0][0];
    expect(messages).toHaveLength(3);
    expect(messages[2]).toEqual({ role: 'user', content: '응\n오늘 날씨 좋네' });
  });

  it('falls back to a calming reply when Claude fails', async () => {
    mockCreate.mockRejectedValue(new Error('overloaded'));

    const result = await service.handleElderUtterance('s1', '오늘 점심 뭐 먹지');

    expect(result.reply).toBeTruthy();
    expect(result.escalated).toBe(false);
    expect(prisma.utterance.create).toHaveBeenNthCalledWith(2, {
      data: expect.objectContaining({ speaker: Speaker.AI, text: result.reply }),
    });
  });

  it('falls back when Claude returns no text', async () => {
    mockCreate.mockResolvedValue({ content: [] });

    const result = await service.handleElderUtterance('s1', '음');

    expect(result.reply).toBeTruthy();
  });

  it('rejects empty text', async () => {
    await expect(service.handleElderUtterance('s1', '   ')).rejects.toBeInstanceOf(BadRequestException);
    expect(prisma.utterance.create).not.toHaveBeenCalled();
  });

  it('rejects an unknown session', async () => {
    prisma.conversationSession.findUnique.mockResolvedValue(null);
    await expect(service.handleElderUtterance('nope', '안녕')).rejects.toBeInstanceOf(NotFoundException);
  });

  it('rejects a session that already ended', async () => {
    prisma.conversationSession.findUnique.mockResolvedValue({ ...activeSession, endedAt: new Date() });
    await expect(service.handleElderUtterance('s1', '안녕')).rejects.toBeInstanceOf(BadRequestException);
    expect(prisma.utterance.create).not.toHaveBeenCalled();
  });
});
