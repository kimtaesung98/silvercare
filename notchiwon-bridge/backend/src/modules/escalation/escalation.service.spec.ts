import { EscalationTriggerType } from '@prisma/client';
import { EscalationService } from './escalation.service';
import { PrismaService } from '../../prisma/prisma.service';

describe('EscalationService', () => {
  const prisma = { escalationEvent: { create: jest.fn() } };
  const service = new EscalationService(prisma as unknown as PrismaService);

  beforeEach(() => jest.clearAllMocks());

  describe('detect', () => {
    it.each([
      ['아까 화장실에서 넘어졌어', EscalationTriggerType.FALL_MENTION],
      ['길에서 쓰러졌었지', EscalationTriggerType.FALL_MENTION],
      ['다리가 너무 아파', EscalationTriggerType.PAIN_COMPLAINT],
      ['머리가 아프네', EscalationTriggerType.PAIN_COMPLAINT],
      ['숨이 차서 힘들어', EscalationTriggerType.PAIN_COMPLAINT],
      ['그냥 죽고 싶어', EscalationTriggerType.SELF_OR_OTHER_HARM],
      ['누가 나를 때리려고 해', EscalationTriggerType.SELF_OR_OTHER_HARM],
    ])('flags "%s" as %s', (text, expected) => {
      expect(service.detect(text)).toBe(expected);
    });

    it.each(['오늘 날씨가 좋네', '손주가 어제 놀러 왔어', '우리 아파트 앞에 꽃이 폈어'])(
      'does not flag "%s"',
      (text) => {
        expect(service.detect(text)).toBeNull();
      },
    );
  });

  it('raiseEscalation records an event for the utterance', async () => {
    prisma.escalationEvent.create.mockResolvedValue({ id: 'ev1' });

    await service.raiseEscalation('s1', 'u1', EscalationTriggerType.FALL_MENTION);

    expect(prisma.escalationEvent.create).toHaveBeenCalledWith({
      data: expect.objectContaining({
        sessionId: 's1',
        utteranceId: 'u1',
        triggerType: EscalationTriggerType.FALL_MENTION,
      }),
    });
  });
});
