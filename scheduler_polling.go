package nautilus

import (
	"context"
	"time"
)

type PollScheduler struct {
	runnerInterval       time.Duration
	skipScheduleInterval time.Duration
	scheduleReader       HookScheduleReader
}

func NewPollScheduler(scheduleReader HookScheduleReader, options ...func(*PollScheduler)) *PollScheduler {
	p := &PollScheduler{
		runnerInterval:       10 * time.Second, // default value
		skipScheduleInterval: 40 * time.Second, // default value
		scheduleReader:       scheduleReader,
	}

	for i := range options {
		options[i](p)
	}

	return p
}

func WithSkipScheduleInterval(skipScheduleInterval time.Duration) func(*PollScheduler) {
	return func(p *PollScheduler) {
		p.skipScheduleInterval = skipScheduleInterval
	}
}

func WithRunnerInterval(runnerInterval time.Duration) func(*PollScheduler) {
	return func(p *PollScheduler) {
		p.runnerInterval = runnerInterval
	}
}

func (p *PollScheduler) Start(ctx context.Context, scheduleCh chan *HookSchedule, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(p.runnerInterval):
			now := time.Now().UTC()
			schedules, err := p.scheduleReader.FindScheduledHookSchedules(ctx)
			if err != nil && errCh != nil {
				errCh <- err
				continue
			}

			for i := range schedules {
				if schedules[i].UpdatedAt != nil && schedules[i].UpdatedAt.After(now.Add(-p.skipScheduleInterval)) {
					continue
				}

				scheduleCh <- schedules[i]
			}
		}
	}
}
