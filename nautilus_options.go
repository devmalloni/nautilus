package nautilus

import (
	"net/http"
	"time"
)

func WithPersister(persister NautilusPersister) func(*Nautilus) {
	return func(n *Nautilus) {
		n.persister = persister
	}
}

func WithHttpClient(httpClient *http.Client) func(*Nautilus) {
	return func(n *Nautilus) {
		n.httpClient = httpClient
	}
}

func WithWorkersCount(workersCount int) func(*Nautilus) {
	return func(n *Nautilus) {
		n.workersCount = workersCount
	}
}

func WithScheduleBufferSize(scheduleBufferSize int) func(*Nautilus) {
	return func(n *Nautilus) {
		n.scheduleBufferSize = scheduleBufferSize
	}
}

func WithSkipScheduleInterval(skipScheduleInterval time.Duration) func(*Nautilus) {
	return func(n *Nautilus) {
		n.skipScheduleInterval = skipScheduleInterval
	}
}

func WithRunnerInterval(runnerInterval time.Duration) func(*Nautilus) {
	return func(n *Nautilus) {
		n.runnerInterval = runnerInterval
	}
}

func WithErrCh(errCh chan<- error) func(*Nautilus) {
	return func(n *Nautilus) {
		n.errCh = errCh
	}
}

func WithJsonSchemaValidator(validator JSchemaValidator) func(*Nautilus) {
	return func(n *Nautilus) {
		n.jsonSchemaValidator = validator
	}
}

func New(options ...func(*Nautilus)) *Nautilus {
	n := &Nautilus{
		jsonSchemaValidator:  NewStandardJsonSchemaValidator(),
		persister:            NewInMemoryPersister(),
		httpClient:           http.DefaultClient,
		workersCount:         5, // default values
		scheduleBufferSize:   100,
		skipScheduleInterval: 40 * time.Second,
		runnerInterval:       10 * time.Second,
		errCh:                nil,
	}

	for i := range options {
		options[i](n)
	}

	return n
}
