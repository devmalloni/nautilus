package nautilus

import (
	"net/http"
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

func WithScheduler(scheduler NautilusScheduler) func(*Nautilus) {
	return func(n *Nautilus) {
		n.scheduler = scheduler
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
		jsonSchemaValidator: NewStandardJsonSchemaValidator(),
		persister:           NewInMemoryPersister(),
		httpClient:          http.DefaultClient,
		workersCount:        5, // default values
		scheduleBufferSize:  100,
		errCh:               nil,
	}

	for i := range options {
		options[i](n)
	}

	// default scheduler
	if n.scheduler == nil {
		n.scheduler = NewPollScheduler(n.persister)
	}

	return n
}
