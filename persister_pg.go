package nautilus

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type SqlPersister struct {
	db *sqlx.DB
}

func WithConnection(db *sqlx.DB) func(p *SqlPersister) error {
	return func(p *SqlPersister) error {
		p.db = db
		return nil
	}
}

func WithDsnConnect(driverName, dsn string) func(p *SqlPersister) error {
	return func(p *SqlPersister) error {
		db, err := sqlx.Connect(driverName, dsn)
		if err != nil {
			return err
		}

		p.db = db
		return nil
	}
}

func NewSQLPersister(options ...func(p *SqlPersister) error) (*SqlPersister, error) {
	p := &SqlPersister{}

	for _, option := range options {
		err := option(p)
		if err != nil {
			return nil, err
		}
	}

	if err := p.db.Ping(); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *SqlPersister) FindHookSchedulesByID(ctx context.Context, id string) (*HookSchedule, []*HookExecution, error) {
	hookSchedule := &HookSchedule{}
	err := p.db.GetContext(ctx, hookSchedule, "SELECT * FROM hook_schedules WHERE id = $1", id)
	if err != nil {
		return nil, nil, err
	}

	hookConfiguration, err := p.FindHookConfigurationByID(ctx, hookSchedule.HookConfigurationID)
	if err != nil {
		return nil, nil, err
	}
	hookSchedule.HookConfiguration = hookConfiguration

	hookExecutions := []*HookExecution{}
	err = p.db.SelectContext(ctx, &hookExecutions, "SELECT * FROM hook_executions WHERE hook_schedule_id = $1", id)
	if err != nil {
		return nil, nil, err
	}

	return hookSchedule, hookExecutions, nil
}

func (p *SqlPersister) FindHookSchedulesOfTag(ctx context.Context, tag HookConfigurationTag) ([]*HookSchedule, error) {
	hookSchedules := []*HookSchedule{}
	err := p.db.SelectContext(ctx, &hookSchedules, "SELECT * FROM hook_schedules WHERE tag = $1", tag)
	if err != nil {
		return nil, err
	}

	return hookSchedules, nil
}

func (p *SqlPersister) FindScheduledHookSchedules(ctx context.Context) ([]*HookSchedule, error) {
	hookSchedules := []*HookSchedule{}
	err := p.db.SelectContext(ctx, &hookSchedules, "SELECT * FROM hook_schedules WHERE status = $1", HookScheduleStatusScheduled)
	if err != nil {
		return nil, err
	}

	return hookSchedules, nil
}

func (p *SqlPersister) WriteHookSchedule(ctx context.Context, c *HookSchedule, e ...*HookExecution) error {
	tx := p.db.MustBeginTx(ctx, nil)
	_, err := tx.NamedExecContext(ctx,
		`INSERT INTO hook_schedules (id, hook_configuration_id, http_request_method, url, payload, status, max_attempt, current_attempt, created_at, updated_at)
			VALUES 					(:id, :hook_configuration_id, :http_request_method, :url, :payload, :status, :max_attempt, :current_attempt, :created_at, :updated_at)
			ON CONFLICT (id) 
			DO UPDATE SET status = excluded.status , current_attempt = excluded.current_attempt, updated_at = excluded.updated_at;`, c)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, execution := range e {
		_, err := tx.NamedExecContext(ctx,
			`INSERT INTO hook_executions (id, hook_schedule_id, response_status, response_payload, created_at) 
			VALUES (:id, :hook_schedule_id, :response_status, :response_payload, :created_at)`, execution)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (p *SqlPersister) FindHookConfiguration(ctx context.Context, hookDefinitionID string, tag HookConfigurationTag) (*HookConfiguration, error) {
	hookConfiguration := &HookConfiguration{}
	err := p.db.GetContext(ctx, hookConfiguration, "SELECT * FROM hook_configurations WHERE hook_definition_id = $1 AND tag = $2", hookDefinitionID, tag)
	if err != nil {
		return nil, err
	}

	hookDefinition, err := p.FindHookDefinitionByID(ctx, hookConfiguration.HookDefinitionID)
	if err != nil {
		return nil, err
	}

	hookConfiguration.HookDefinition = hookDefinition

	return hookConfiguration, nil
}

func (p *SqlPersister) FindHookConfigurationByID(ctx context.Context, id string) (*HookConfiguration, error) {
	hookConfiguration := &HookConfiguration{}
	err := p.db.GetContext(ctx, hookConfiguration, "SELECT * FROM hook_configurations WHERE id = $1", id)
	if err != nil {
		return nil, err
	}

	hookDefinition, err := p.FindHookDefinitionByID(ctx, hookConfiguration.HookDefinitionID)
	if err != nil {
		return nil, err
	}

	hookConfiguration.HookDefinition = hookDefinition

	return hookConfiguration, nil
}

func (p *SqlPersister) FindHookConfigurationsByTag(ctx context.Context, tag HookConfigurationTag) ([]*HookConfiguration, error) {
	hookConfigurations := []*HookConfiguration{}
	err := p.db.SelectContext(ctx, &hookConfigurations, "SELECT * FROM hook_configurations WHERE tag = $1", tag)
	if err != nil {
		return nil, err
	}

	return hookConfigurations, nil
}

func (p *SqlPersister) FindHookConfigurations(ctx context.Context) ([]*HookConfiguration, error) {
	hookConfigurations := []*HookConfiguration{}
	err := p.db.SelectContext(ctx, &hookConfigurations, "SELECT * FROM hook_configurations")
	if err != nil {
		return nil, err
	}

	return hookConfigurations, nil
}

func (p *SqlPersister) WriteHookConfiguration(ctx context.Context, c *HookConfiguration) error {
	_, err := p.db.NamedExecContext(ctx,
		`INSERT INTO hook_configurations (id, hook_definition_id, url, tag, client_secret, client_rsa_private_key, created_at)
			VALUES (:id, :hook_definition_id, :url, :tag, :client_secret, :client_rsa_private_key, :created_at)
			ON CONFLICT (id) 
			DO UPDATE SET url = excluded.url, tag = excluded.tag, client_secret = excluded.client_secret, client_rsa_private_key = excluded.client_rsa_private_key, created_at = excluded.created_at;`, c)
	if err != nil {
		return err
	}

	return nil
}

func (p *SqlPersister) FindHookDefinitionByID(ctx context.Context, id string) (*HookDefinition, error) {
	hookDefinition := &HookDefinition{}
	err := p.db.GetContext(ctx, hookDefinition, "SELECT * FROM hook_definitions WHERE id = $1", id)
	if err != nil {
		return nil, err
	}

	return hookDefinition, nil
}

func (p *SqlPersister) FindHookDefinitions(ctx context.Context) ([]*HookDefinition, error) {
	hookDefinitions := []*HookDefinition{}
	err := p.db.SelectContext(ctx, &hookDefinitions, "SELECT * FROM hook_definitions")
	if err != nil {
		return nil, err
	}

	return hookDefinitions, nil
}

func (p *SqlPersister) WriteHookDefinitions(ctx context.Context, d ...*HookDefinition) error {
	tx := p.db.MustBeginTx(ctx, nil)
	for _, definition := range d {
		_, err := tx.NamedExecContext(ctx,
			`INSERT INTO hook_definitions (id, name, description, payload_scheme, http_request_method, total_attempts)
				VALUES (:id, :name, :description, :payload_scheme, :http_request_method, :total_attempts)
				ON CONFLICT (id) 
				DO UPDATE SET name = excluded.name, description = excluded.description, payload_scheme = excluded.payload_scheme, http_request_method = excluded.http_request_method,
					total_attempts = excluded.total_attempts;`, definition)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	err := tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
