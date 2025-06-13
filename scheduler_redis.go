package nautilus

import (
	"context"
	"time"
)

type RedisScheduler struct {
	runnerInterval       time.Duration
	skipScheduleInterval time.Duration
	scheduleReader       HookScheduleReader
}

func NewRedisScheduler(scheduleReader HookScheduleReader, options ...func(*RedisScheduler)) *RedisScheduler {
	p := &RedisScheduler{
		runnerInterval:       10 * time.Second, // default value
		skipScheduleInterval: 40 * time.Second, // default value
		scheduleReader:       scheduleReader,
	}

	for i := range options {
		options[i](p)
	}

	return p
}

func (p *RedisScheduler) Start(ctx context.Context, scheduleCh chan *HookSchedule, errCh chan<- error) {
	go p.startScheduler(ctx, errCh)

	p.startConsumer(ctx, scheduleCh)
}

func (p *RedisScheduler) startScheduler(ctx context.Context, errCh chan<- error) {

}

func (p *RedisScheduler) startConsumer(ctx context.Context, scheduleCh chan *HookSchedule) {

}
