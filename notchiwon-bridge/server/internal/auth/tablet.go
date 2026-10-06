package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/kimtaesung98/silvercare/notchiwon-bridge/server/internal/db"
)

// ErrUnknownDevice means the token matches no active elder tablet.
var ErrUnknownDevice = errors.New("unknown or revoked device")

// LookupTablet finds the elder tablet a device token belongs to and records
// that it was seen. Other errors are database errors.
func LookupTablet(ctx context.Context, q *db.Queries, token string) (Tablet, error) {
	d, err := q.GetActiveDeviceByTokenHash(ctx, HashDeviceToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return Tablet{}, ErrUnknownDevice
	}
	if err != nil {
		return Tablet{}, fmt.Errorf("read device: %w", err)
	}
	if d.Kind != "ELDER_TABLET" || d.ElderID == nil {
		return Tablet{}, ErrUnknownDevice
	}
	if err := q.TouchDevice(ctx, d.ID); err != nil {
		return Tablet{}, fmt.Errorf("touch device: %w", err)
	}
	return Tablet{DeviceID: d.ID, ElderID: *d.ElderID}, nil
}
