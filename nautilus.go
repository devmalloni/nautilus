package nautilus

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/devmalloni/nautilus/x"
)

const (
	Global HookConfigurationTag = "global"
)

type (
	NautilusScheduler interface {
		Start(ctx context.Context, scheduleCh chan *HookSchedule, errCh chan<- error)
	}
	Nautilus struct {
		jsonSchemaValidator JSchemaValidator
		persister           NautilusPersister
		httpClient          *http.Client
		workersCount        int
		scheduleBufferSize  int
		scheduler           NautilusScheduler
		errCh               chan<- error
	}
)

func (p *Nautilus) Run(ctx context.Context) {
	reportError := func(errCh chan<- error, err error) {
		if errCh != nil {
			errCh <- err
		}
	}

	worker := func(ctx context.Context, scheduleCh chan *HookSchedule, errCh chan<- error) {
		for schedule := range scheduleCh {
			err := p.executeSchedule(ctx, schedule.ID)
			if err != nil {
				reportError(errCh, err)
			}
		}
	}

	scheduleCh := make(chan *HookSchedule, p.scheduleBufferSize)
	defer close(scheduleCh)

	// start workers
	for i := 0; i < p.workersCount; i++ {
		go worker(ctx, scheduleCh, p.errCh)
	}

	p.scheduler.Start(ctx, scheduleCh, p.errCh)
}

// TrySchedule is a convenience method that checks if a hook configuration exists
// before scheduling a hook. If the configuration does not exist, it returns nil.
func (p *Nautilus) TrySchedule(ctx context.Context,
	id *string,
	hookDefinitionID string,
	tag HookConfigurationTag,
	payload json.RawMessage) error {
	_, err := p.persister.FindHookConfiguration(ctx, hookDefinitionID, tag)
	if err == ErrNotFound {
		return nil
	}

	if err != nil {
		return err
	}

	_, err = p.ScheduleJSON(ctx, id, hookDefinitionID, tag, payload)
	if err != nil {
		return err
	}

	return nil
}

func (p *Nautilus) TryScheduleJSON(ctx context.Context,
	id *string,
	hookDefinitionID string,
	tag HookConfigurationTag,
	payload any) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return p.TrySchedule(ctx, id, hookDefinitionID, tag, jsonPayload)
}

func (p *Nautilus) MustScheduleJSON(ctx context.Context,
	id *string,
	hookDefinitionID string,
	tag HookConfigurationTag,
	payload any) *HookSchedule {
	schedule, err := p.ScheduleJSON(ctx, id, hookDefinitionID, tag, payload)
	if err != nil {
		panic(err)
	}

	return schedule
}

func (p *Nautilus) ScheduleJSON(ctx context.Context,
	id *string,
	hookDefinitionID string,
	tag HookConfigurationTag,
	payload any) (*HookSchedule, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return p.Schedule(ctx, id, hookDefinitionID, tag, jsonPayload)
}

func (p *Nautilus) MustSchedule(ctx context.Context,
	id *string,
	hookDefinitionID string,
	tag HookConfigurationTag,
	payload json.RawMessage) *HookSchedule {
	schedule, err := p.Schedule(ctx, id, hookDefinitionID, tag, payload)
	if err != nil {
		panic(err)
	}

	return schedule
}

func (p *Nautilus) Schedule(ctx context.Context,
	id *string,
	hookDefinitionID string,
	tag HookConfigurationTag,
	payload json.RawMessage) (*HookSchedule, error) {
	configuration, err := p.persister.FindHookConfiguration(ctx, hookDefinitionID, tag)
	if err != nil {
		return nil, err
	}

	var scheduleID string
	if id != nil {
		schedule, _, err := p.FindScheduleByID(ctx, *id)
		if err != nil && err != ErrNotFound {
			return nil, err
		}
		if schedule != nil {
			return nil, errors.New("there already is a schedule with this ID")
		}
		scheduleID = *id
	} else {
		scheduleID = x.NewUUIDStr()
	}

	schedule, err := configuration.Schedule(scheduleID, payload, p.jsonSchemaValidator)
	if err != nil {
		return nil, err
	}

	err = p.persister.WriteHookSchedule(ctx, schedule)
	if err != nil {
		return nil, err
	}

	return schedule, nil
}

func (p *Nautilus) ScheduleAndExecute(ctx context.Context,
	id *string,
	hookDefinitionID string,
	tag HookConfigurationTag,
	payload json.RawMessage) error {
	schedule, err := p.Schedule(ctx, id, hookDefinitionID, tag, payload)
	if err != nil {
		return err
	}

	err = p.executeSchedule(ctx, schedule.ID)
	if err != nil {
		return err
	}

	return nil
}

func (p *Nautilus) executeSchedule(ctx context.Context, scheduleID string) error {
	schedule, _, err := p.FindScheduleByID(ctx, scheduleID)
	if err != nil {
		return err
	}

	execution, err := schedule.Execute(ctx, x.NewUUIDStr(), p.httpClient)
	if err != nil {
		return err
	}

	err = p.persister.WriteHookSchedule(ctx, schedule, execution)
	if err != nil {
		return err
	}

	return nil
}

// To User
func (p *Nautilus) RetryScheduleByID(ctx context.Context, scheduleID string) error {
	schedule, _, err := p.persister.FindHookSchedulesByID(ctx, scheduleID)
	if err != nil {
		return err
	}

	err = p.executeSchedule(ctx, schedule.ID)
	if err != nil {
		return err
	}

	return nil
}

func (p *Nautilus) ListSchedulesOfTag(ctx context.Context, tag HookConfigurationTag) ([]*HookSchedule, error) {
	return p.persister.FindHookSchedulesOfTag(ctx, tag)
}

func (p *Nautilus) FindScheduleByID(ctx context.Context, scheduleID string) (*HookSchedule, []*HookExecution, error) {
	return p.persister.FindHookSchedulesByID(ctx, scheduleID)
}

func (p *Nautilus) ListConfigurationsOfTag(ctx context.Context, tag HookConfigurationTag) ([]*HookConfiguration, error) {
	return p.persister.FindHookConfigurationsByTag(ctx, tag)
}

func (p *Nautilus) ListDefinitions(ctx context.Context) ([]*HookDefinition, error) {
	return p.persister.FindHookDefinitions(ctx)
}

func (p *Nautilus) ListAllConfigurations(ctx context.Context) ([]*HookConfiguration, error) {
	return p.persister.FindHookConfigurations(ctx)
}

func (p *Nautilus) CreateConfigurationFromDefinition(ctx context.Context,
	definitionID string,
	url string,
	tag HookConfigurationTag,
	secret *string) (*HookConfiguration, error) {
	definition, err := p.persister.FindHookDefinitionByID(ctx, definitionID)
	if err != nil {
		return nil, err
	}

	configuration, err := definition.CreateConfiguration(x.NewUUIDStr(), url, tag, secret)
	if err != nil {
		return nil, err
	}

	err = configuration.GeneratePrivateKey(false)
	if err != nil {
		return nil, err
	}

	err = p.persister.WriteHookConfiguration(ctx, configuration)
	if err != nil {
		return nil, err
	}

	return configuration, nil
}

func (p *Nautilus) RegisterConfigurations(ctx context.Context, configurations ...*HookConfiguration) error {
	for i := range configurations {
		definition, err := p.persister.FindHookDefinitionByID(ctx, configurations[i].HookDefinitionID)
		if err != nil {
			return err
		}
		configurations[i].HookDefinition = definition

		if err := configurations[i].IsValid(); err != nil {
			return err
		}

		err = p.persister.WriteHookConfiguration(ctx, configurations[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Nautilus) RegisterDefinitions(ctx context.Context, definitions ...*HookDefinition) error {
	for i := range definitions {
		if err := definitions[i].IsValid(); err != nil {
			return err
		}

		err := p.persister.WriteHookDefinitions(ctx, definitions[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func ID(id string) *string {
	if id == "" {
		uid := x.NewUUIDStr()
		return &uid
	}

	return &id
}
