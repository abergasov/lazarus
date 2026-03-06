package push

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Subscription struct {
	ID       uuid.UUID `db:"id"`
	UserID   uuid.UUID `db:"user_id"`
	Endpoint string    `db:"endpoint"`
	P256DH   string    `db:"p256dh"`
	AuthKey  string    `db:"auth_key"`
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Subscribe(ctx context.Context, userID uuid.UUID, endpoint, p256dh, authKey string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth_key)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (endpoint) DO NOTHING
	`, userID, endpoint, p256dh, authKey)
	return err
}

// Send sends a push notification to all subscriptions for a user.
// In production, this should use the Web Push protocol (VAPID).
// For MVP, this logs intent and can be wired to a library like SheriffMarshal/webpush-go.
func (s *Service) Send(ctx context.Context, userID uuid.UUID, message string) {
	var subs []Subscription
	_ = s.db.SelectContext(ctx, &subs, `SELECT * FROM push_subscriptions WHERE user_id = $1`, userID)
	// TODO: send Web Push notification to each subscription
	_ = message
}
