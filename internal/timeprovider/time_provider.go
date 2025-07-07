package timeprovider

import "time"

type TimeProvider interface {
	Now() time.Time
	IsExpired(expiryTime int64) bool
}

type RealTimeProvider struct{}

func (rtp *RealTimeProvider) Now() time.Time {
	return time.Now()
}

func (rtp *RealTimeProvider) IsExpired(expiryTime int64) bool {
	return rtp.Now().After(time.Unix(expiryTime, 0))
}

func DefaultTimeProvider() TimeProvider {
	return &RealTimeProvider{}
}

// Ensure RealTimeProvider implements TimeProvider
var _ TimeProvider = (*RealTimeProvider)(nil)
