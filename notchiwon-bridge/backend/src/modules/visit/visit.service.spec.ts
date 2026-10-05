import { Logger } from '@nestjs/common';
import { VisitStatus } from '@prisma/client';
import { VisitService } from './visit.service';

describe('VisitService.handleLocationUpdate', () => {
  const input = { caregiverId: 'c1', visitId: 'v1', latitude: 37.5, longitude: 127.0 };

  // Minimal in-memory stand-in for prisma.visit that keeps the stored status.
  function setup(initialStatus: VisitStatus) {
    const row = { id: 'v1', status: initialStatus, etaCurrent: null as Date | null };
    const prisma = {
      visit: {
        findUnique: jest.fn(async () => ({ ...row })),
        update: jest.fn(async ({ data }: { data: Partial<typeof row> }) => {
          Object.assign(row, data);
          return { ...row };
        }),
      },
    };
    const service = new VisitService(prisma as any);
    const triggerLog = jest.spyOn(Logger.prototype, 'log').mockImplementation(() => undefined);
    return { service, row, triggerLog };
  }

  afterEach(() => jest.restoreAllMocks());

  it('fires the session trigger once when the visit is en route', async () => {
    const { service, row, triggerLog } = setup(VisitStatus.EN_ROUTE);

    await service.handleLocationUpdate(input);

    expect(triggerLog).toHaveBeenCalledTimes(1);
    expect(row.status).toBe(VisitStatus.EN_ROUTE);
  });

  it('does not fire again or reset status while a session is already active', async () => {
    const { service, row, triggerLog } = setup(VisitStatus.SESSION_ACTIVE);

    await service.handleLocationUpdate(input);
    await service.handleLocationUpdate(input);

    expect(triggerLog).not.toHaveBeenCalled();
    expect(row.status).toBe(VisitStatus.SESSION_ACTIVE);
    expect(row.etaCurrent).toBeInstanceOf(Date);
  });

  it.each([VisitStatus.COMPLETED, VisitStatus.CANCELLED])(
    'leaves a %s visit alone',
    async (status) => {
      const { service, row, triggerLog } = setup(status);

      await service.handleLocationUpdate(input);

      expect(triggerLog).not.toHaveBeenCalled();
      expect(row.status).toBe(status);
    },
  );
});
