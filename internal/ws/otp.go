package ws

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Authentication for websocket connection
// Client need to login first to get the OTP from server
// Then client connect to websocket with that OTP in query param
// If OTP valid, client connected to websocket

type OTP struct {
	CreatedAt time.Time
	Key       string `json:"key"`
}

type RetentionMap map[string]OTP

func NewRetentionMap(ctx context.Context, retentionPeriod time.Duration) RetentionMap {
	rm := make(RetentionMap)

	go rm.Retention(ctx, retentionPeriod)

	return rm
}

func (rm RetentionMap) NewOTP() OTP {
	otp := OTP{
		Key:       uuid.NewString(),
		CreatedAt: time.Now(),
	}

	rm[otp.Key] = otp

	return otp
}

func (rm RetentionMap) VerifyOTP(key string) bool {
	if _, ok := rm[key]; !ok {
		return false // otp is not crreated
	}

	delete(rm, key)
	return true
}

// Retention check every tick to delete the otp that too long from RetentionMap
func (rm RetentionMap) Retention(ctx context.Context, retentionPeriod time.Duration) {
	ticker := time.NewTicker(400 * time.Microsecond)

	for {
		select {
		case <-ticker.C:
			for _, otp := range rm {
				if otp.CreatedAt.Add(retentionPeriod).Before(time.Now()) {
					delete(rm, otp.Key)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
