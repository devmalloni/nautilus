package x

import (
	"time"

	"github.com/google/uuid"
)

func NewUUIDStr() string {
	return uuid.New().String()
}

func NilTime(t time.Time) *time.Time {
	return &t
}
