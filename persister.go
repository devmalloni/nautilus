package nautilus

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("record not found")
)

type (
	HookScheduleReader interface {
		FindHookSchedulesByID(ctx context.Context, id string) (*HookSchedule, []*HookExecution, error)
		FindHookSchedulesOfTag(ctx context.Context, tag HookConfigurationTag) ([]*HookSchedule, error)
		FindScheduledHookSchedules(ctx context.Context) ([]*HookSchedule, error)
	}

	HookScheduleWriter interface {
		WriteHookSchedule(ctx context.Context, c *HookSchedule, e ...*HookExecution) error
	}

	HookConfigurationReader interface {
		FindHookConfiguration(ctx context.Context, hookDefinitionID string, tag HookConfigurationTag) (*HookConfiguration, error)
		FindHookConfigurationsByTag(ctx context.Context, tag HookConfigurationTag) ([]*HookConfiguration, error)
		FindHookConfigurations(ctx context.Context) ([]*HookConfiguration, error)
	}

	HookConfigurationWriter interface {
		WriteHookConfiguration(ctx context.Context, c *HookConfiguration) error
	}

	HookDefinitionReader interface {
		FindHookDefinitionByID(ctx context.Context, id string) (*HookDefinition, error)
		FindHookDefinitions(ctx context.Context) ([]*HookDefinition, error)
	}

	HookDefinitionWriter interface {
		WriteHookDefinitions(ctx context.Context, d ...*HookDefinition) error
	}

	NautilusPersister interface {
		HookScheduleReader
		HookScheduleWriter

		HookConfigurationReader
		HookConfigurationWriter

		HookDefinitionReader
		HookDefinitionWriter
	}
)
