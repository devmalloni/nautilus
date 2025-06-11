package nautilus

import (
	"context"
	"sync"
)

type InMemoryPersister struct {
	l              *sync.Mutex
	definitions    map[string]*HookDefinition
	schedules      map[string]*HookSchedule
	configurations map[string]*HookConfiguration
	executions     map[string][]*HookExecution
}

func NewInMemoryPersister() *InMemoryPersister {
	return &InMemoryPersister{
		l:              &sync.Mutex{},
		definitions:    make(map[string]*HookDefinition),
		schedules:      make(map[string]*HookSchedule),
		configurations: make(map[string]*HookConfiguration),
		executions:     make(map[string][]*HookExecution),
	}
}

func (p *InMemoryPersister) FindHookSchedulesByID(ctx context.Context, id string) (*HookSchedule, []*HookExecution, error) {
	p.l.Lock()
	defer p.l.Unlock()
	s, ok := p.schedules[id]
	if !ok {
		return nil, nil, ErrNotFound
	}
	executions := p.executions[id]

	return s, executions, nil
}

func (p *InMemoryPersister) FindHookSchedulesOfTag(ctx context.Context, tag HookConfigurationTag) ([]*HookSchedule, error) {
	p.l.Lock()
	defer p.l.Unlock()

	var res []*HookSchedule
	for _, v := range p.schedules {
		if v.HookConfiguration.Tag == tag {
			res = append(res, v)
		}
	}

	return res, nil
}

func (p *InMemoryPersister) FindScheduledHookSchedules(ctx context.Context) ([]*HookSchedule, error) {
	p.l.Lock()
	defer p.l.Unlock()

	var res []*HookSchedule
	for _, v := range p.schedules {
		if v.Status == HookScheduleStatusScheduled {
			res = append(res, v)
		}
	}

	return res, nil
}

func (p *InMemoryPersister) WriteHookSchedule(ctx context.Context, c *HookSchedule, e ...*HookExecution) error {
	p.l.Lock()
	defer p.l.Unlock()

	p.schedules[c.ID] = c

	p.executions[c.ID] = append(p.executions[c.ID], e...)

	return nil
}

func (p *InMemoryPersister) FindHookConfiguration(ctx context.Context, hookDefinitionID string, tag HookConfigurationTag) (*HookConfiguration, error) {
	p.l.Lock()
	defer p.l.Unlock()

	for _, v := range p.configurations {
		if v.HookDefinitionID == hookDefinitionID && v.Tag == tag {
			return v, nil
		}
	}

	return nil, ErrNotFound
}

func (p *InMemoryPersister) FindHookConfigurationsByTag(ctx context.Context, tag HookConfigurationTag) ([]*HookConfiguration, error) {
	p.l.Lock()
	defer p.l.Unlock()

	var res []*HookConfiguration
	for _, v := range p.configurations {
		if v.Tag == tag {
			res = append(res, v)
		}
	}

	return res, nil
}

func (p *InMemoryPersister) FindHookConfigurations(ctx context.Context) ([]*HookConfiguration, error) {
	p.l.Lock()
	defer p.l.Unlock()

	var res []*HookConfiguration
	for _, v := range p.configurations {
		res = append(res, v)
	}

	return res, nil
}

func (p *InMemoryPersister) WriteHookConfiguration(ctx context.Context, c *HookConfiguration) error {
	p.l.Lock()
	defer p.l.Unlock()

	p.configurations[c.ID] = c

	return nil
}

func (p *InMemoryPersister) FindHookDefinitionByID(ctx context.Context, id string) (*HookDefinition, error) {
	p.l.Lock()
	defer p.l.Unlock()
	d, ok := p.definitions[id]
	if !ok {
		return nil, ErrNotFound
	}

	return d, nil
}

func (p *InMemoryPersister) FindHookDefinitions(ctx context.Context) ([]*HookDefinition, error) {
	p.l.Lock()
	defer p.l.Unlock()

	var res []*HookDefinition
	for _, v := range p.definitions {
		res = append(res, v)
	}

	return res, nil
}

func (p *InMemoryPersister) WriteHookDefinitions(ctx context.Context, d ...*HookDefinition) error {
	p.l.Lock()
	defer p.l.Unlock()

	for _, v := range d {
		p.definitions[v.ID] = v

	}

	return nil
}
