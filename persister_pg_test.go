package nautilus

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/devmalloni/nautilus/x"
	"github.com/jmoiron/sqlx"
)

func mustCreateTestPersister(t *testing.T) (*SqlPersister, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	sqlxdb := sqlx.NewDb(db, "postgres")

	return &SqlPersister{
			db: sqlxdb,
		},
		mock,
		db.Close
}

func TestSqlPersister_FindHookSchedulesByID(t *testing.T) {
	persister, mock, close := mustCreateTestPersister(t)
	defer close()

	id := "test-id"

	now := time.Now().UTC()
	expectedResult := &HookSchedule{
		ID:                  id,
		HookConfigurationID: "hook-config-id",
		HttpRequestMethod:   POST,
		URL:                 "http://example.com",
		Payload:             json.RawMessage(`{"key":"value"}`),
		Status:              "pending",
		MaxAttempt:          3,
		CurrentAttempt:      0,
		CreatedAt:           now,
		UpdatedAt:           &now,
	}

	mock.ExpectQuery(`SELECT (.+) FROM hook_schedules`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"hook_configuration_id",
			"http_request_method",
			"url",
			"payload",
			"status",
			"max_attempt",
			"current_attempt",
			"created_at",
			"updated_at",
		}).AddRow(
			expectedResult.ID,
			expectedResult.HookConfigurationID,
			expectedResult.HttpRequestMethod,
			expectedResult.URL,
			expectedResult.Payload,
			expectedResult.Status,
			expectedResult.MaxAttempt,
			expectedResult.CurrentAttempt,
			expectedResult.CreatedAt,
			expectedResult.UpdatedAt,
		))

	expectedConfiguration := &HookConfiguration{
		ID:                  expectedResult.HookConfigurationID,
		HookDefinitionID:    "hook-definition-id",
		Tag:                 "example-tag",
		URL:                 "http://example.com/config",
		ClientSecret:        nil,
		ClientRSAPrivateKey: nil,
		CreatedAt:           now,
	}

	mock.ExpectQuery(`SELECT (.+) FROM hook_configurations`).
		WithArgs(expectedResult.HookConfigurationID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"hook_definition_id",
			"tag",
			"url",
			"client_secret",
			"client_rsa_private_key",
			"created_at",
		}).AddRow(
			expectedConfiguration.ID,
			expectedConfiguration.HookDefinitionID,
			expectedConfiguration.Tag,
			expectedConfiguration.URL,
			expectedConfiguration.ClientSecret,
			expectedConfiguration.ClientRSAPrivateKey,
			expectedConfiguration.CreatedAt,
		))

	expectedDefinition := &HookDefinition{
		ID:                "hook-definition-id",
		Description:       "Example Hook Definition",
		PayloadScheme:     json.RawMessage(`{"type":"object","properties":{"key":{"type":"string"}}}`),
		HttpRequestMethod: POST,
		TotalAttempts:     5,
	}

	mock.ExpectQuery(`SELECT (.+) FROM hook_definitions`).
		WithArgs(expectedConfiguration.HookDefinitionID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"name",
			"description",
			"payload_scheme",
			"http_request_method",
			"total_attempts",
		}).AddRow(
			expectedDefinition.ID,
			expectedDefinition.Name,
			expectedDefinition.Description,
			expectedDefinition.PayloadScheme,
			expectedDefinition.HttpRequestMethod,
			expectedDefinition.TotalAttempts,
		))

	expectedExecution := &HookExecution{
		ID:              "execution-id",
		HookScheduleID:  id,
		ResponseStatus:  http.StatusOK,
		ResponsePayload: x.NullString(`{"response":"ok"}`),
		CreatedAt:       now,
	}

	mock.ExpectQuery(`SELECT (.+) FROM hook_executions`).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"hook_schedule_id",
			"response_status",
			"response_payload",
			"created_at",
		}).AddRow(
			expectedExecution.ID,
			expectedExecution.HookScheduleID,
			expectedExecution.ResponseStatus,
			expectedExecution.ResponsePayload,
			expectedExecution.CreatedAt,
		))

	schedule, executions, err := persister.FindHookSchedulesByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schedule == nil {
		t.Fatalf("expected result to be non-nil, got nil")
	}

	if len(executions) != 1 {
		t.Fatalf("expected 1 execution, got %d", len(executions))
	}
}

func TestSqlPersister_FindHookSchedulesOfTag(t *testing.T) {
	persister, mock, close := mustCreateTestPersister(t)
	defer close()

	tag := "tag-id"

	now := time.Now().UTC()
	expectedResult := &HookSchedule{
		ID:                  "schedule-id",
		HookConfigurationID: "hook-config-id",
		HttpRequestMethod:   POST,
		URL:                 "http://example.com",
		Payload:             json.RawMessage(`{"key":"value"}`),
		Status:              "pending",
		MaxAttempt:          3,
		CurrentAttempt:      0,
		CreatedAt:           now,
		UpdatedAt:           &now,
	}

	mock.ExpectQuery(`SELECT (.+) FROM hook_schedules`).
		WithArgs(tag).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"hook_configuration_id",
			"http_request_method",
			"url",
			"payload",
			"status",
			"max_attempt",
			"current_attempt",
			"created_at",
			"updated_at",
		}).AddRow(
			expectedResult.ID,
			expectedResult.HookConfigurationID,
			expectedResult.HttpRequestMethod,
			expectedResult.URL,
			expectedResult.Payload,
			expectedResult.Status,
			expectedResult.MaxAttempt,
			expectedResult.CurrentAttempt,
			expectedResult.CreatedAt,
			expectedResult.UpdatedAt,
		))

	res, err := persister.FindHookSchedulesOfTag(context.Background(), HookConfigurationTag(tag))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
}

func TestSqlPersister_FindScheduledHookSchedules(t *testing.T) {
	persister, mock, close := mustCreateTestPersister(t)
	defer close()

	now := time.Now().UTC()
	expectedResult := &HookSchedule{
		ID:                  "schedule-id",
		HookConfigurationID: "hook-config-id",
		HttpRequestMethod:   POST,
		URL:                 "http://example.com",
		Payload:             json.RawMessage(`{"key":"value"}`),
		Status:              "pending",
		MaxAttempt:          3,
		CurrentAttempt:      0,
		CreatedAt:           now,
		UpdatedAt:           &now,
	}

	mock.ExpectQuery(`SELECT (.+) FROM hook_schedules`).
		WithArgs(HookScheduleStatusScheduled).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"hook_configuration_id",
			"http_request_method",
			"url",
			"payload",
			"status",
			"max_attempt",
			"current_attempt",
			"created_at",
			"updated_at",
		}).AddRow(
			expectedResult.ID,
			expectedResult.HookConfigurationID,
			expectedResult.HttpRequestMethod,
			expectedResult.URL,
			expectedResult.Payload,
			expectedResult.Status,
			expectedResult.MaxAttempt,
			expectedResult.CurrentAttempt,
			expectedResult.CreatedAt,
			expectedResult.UpdatedAt,
		))

	res, err := persister.FindScheduledHookSchedules(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
}

func TestSqlPersister_WriteHookSchedule(t *testing.T) {
	persister, mock, close := mustCreateTestPersister(t)
	defer close()

	now := time.Now().UTC()
	schedule := &HookSchedule{
		ID:                  "schedule-id",
		HookConfigurationID: "hook-config-id",
		HttpRequestMethod:   POST,
		URL:                 "http://example.com",
		Payload:             json.RawMessage(`{"key":"value"}`),
		Status:              "pending",
		MaxAttempt:          3,
		CurrentAttempt:      0,
		CreatedAt:           now,
		UpdatedAt:           &now,
	}

	execution := &HookExecution{
		ID:              "execution-id",
		HookScheduleID:  schedule.ID,
		ResponseStatus:  http.StatusOK,
		RequestPayload:  x.NullString(`{"response":"ok"}`),
		ResponsePayload: x.NullString(`{"response":"ok"}`),
		CreatedAt:       now,
	}

	mock.ExpectBegin()

	mock.ExpectExec(`INSERT INTO hook_schedules`).
		WithArgs(schedule.ID,
			schedule.HookConfigurationID,
			schedule.HttpRequestMethod,
			schedule.URL,
			schedule.Payload,
			schedule.Status,
			schedule.MaxAttempt,
			schedule.CurrentAttempt,
			schedule.HideExecutionMetadata,
			schedule.CreatedAt,
			schedule.UpdatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO hook_executions`).
		WithArgs(
			execution.ID,
			execution.HookScheduleID,
			execution.ResponseStatus,
			execution.RequestPayload,
			execution.ResponsePayload,
			execution.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err := persister.WriteHookSchedule(context.Background(), schedule, execution)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSqlPersister_FindHookConfiguration(t *testing.T) {
	persister, mock, close := mustCreateTestPersister(t)
	defer close()

	expectedConfiguration := &HookConfiguration{
		ID:                  "config-id",
		HookDefinitionID:    "hook-definition-id",
		Tag:                 "example-tag",
		URL:                 "http://example.com/config",
		ClientSecret:        nil,
		ClientRSAPrivateKey: nil,
		CreatedAt:           time.Now(),
	}

	mock.ExpectQuery(`SELECT (.+) FROM hook_configurations`).
		WithArgs(expectedConfiguration.ID, expectedConfiguration.Tag).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"hook_definition_id",
			"tag",
			"url",
			"client_secret",
			"client_rsa_private_key",
			"created_at",
		}).AddRow(
			expectedConfiguration.ID,
			expectedConfiguration.HookDefinitionID,
			expectedConfiguration.Tag,
			expectedConfiguration.URL,
			expectedConfiguration.ClientSecret,
			expectedConfiguration.ClientRSAPrivateKey,
			expectedConfiguration.CreatedAt,
		))

	expectedDefinition := &HookDefinition{
		ID:                "hook-definition-id",
		Description:       "Example Hook Definition",
		PayloadScheme:     json.RawMessage(`{"type":"object","properties":{"key":{"type":"string"}}}`),
		HttpRequestMethod: POST,
		TotalAttempts:     5,
	}

	mock.ExpectQuery(`SELECT (.+) FROM hook_definitions`).
		WithArgs(expectedConfiguration.HookDefinitionID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"name",
			"description",
			"payload_scheme",
			"http_request_method",
			"total_attempts",
		}).AddRow(
			expectedDefinition.ID,
			expectedDefinition.Name,
			expectedDefinition.Description,
			expectedDefinition.PayloadScheme,
			expectedDefinition.HttpRequestMethod,
			expectedDefinition.TotalAttempts,
		))

	config, err := persister.FindHookConfiguration(context.Background(), expectedConfiguration.ID, HookConfigurationTag(expectedConfiguration.Tag))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config == nil {
		t.Fatalf("expected configuration to be non-nil, got nil")
	}
}

func TestSqlPersister_FindHookConfigurationByID(t *testing.T) {
	persister, mock, close := mustCreateTestPersister(t)
	defer close()

	expectedConfiguration := &HookConfiguration{
		ID:                  "config-id",
		HookDefinitionID:    "hook-definition-id",
		Tag:                 "example-tag",
		URL:                 "http://example.com/config",
		ClientSecret:        nil,
		ClientRSAPrivateKey: nil,
		CreatedAt:           time.Now(),
	}

	mock.ExpectQuery(`SELECT (.+) FROM hook_configurations`).
		WithArgs(expectedConfiguration.ID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"hook_definition_id",
			"tag",
			"url",
			"client_secret",
			"client_rsa_private_key",
			"created_at",
		}).AddRow(
			expectedConfiguration.ID,
			expectedConfiguration.HookDefinitionID,
			expectedConfiguration.Tag,
			expectedConfiguration.URL,
			expectedConfiguration.ClientSecret,
			expectedConfiguration.ClientRSAPrivateKey,
			expectedConfiguration.CreatedAt,
		))

	expectedDefinition := &HookDefinition{
		ID:                "hook-definition-id",
		Description:       "Example Hook Definition",
		PayloadScheme:     json.RawMessage(`{"type":"object","properties":{"key":{"type":"string"}}}`),
		HttpRequestMethod: POST,
		TotalAttempts:     5,
	}

	mock.ExpectQuery(`SELECT (.+) FROM hook_definitions`).
		WithArgs(expectedConfiguration.HookDefinitionID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"name",
			"description",
			"payload_scheme",
			"http_request_method",
			"total_attempts",
		}).AddRow(
			expectedDefinition.ID,
			expectedDefinition.Name,
			expectedDefinition.Description,
			expectedDefinition.PayloadScheme,
			expectedDefinition.HttpRequestMethod,
			expectedDefinition.TotalAttempts,
		))

	config, err := persister.FindHookConfigurationByID(context.Background(), expectedConfiguration.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config == nil {
		t.Fatalf("expected configuration to be non-nil, got nil")
	}
}

func TestSqlPersister_FindHookConfigurationsByTag(t *testing.T) {
	persister, mock, close := mustCreateTestPersister(t)
	defer close()

	expectedConfiguration := &HookConfiguration{
		ID:                  "config-id",
		HookDefinitionID:    "hook-definition-id",
		Tag:                 "example-tag",
		URL:                 "http://example.com/config",
		ClientSecret:        nil,
		ClientRSAPrivateKey: nil,
		CreatedAt:           time.Now(),
	}

	mock.ExpectQuery(`SELECT (.+) FROM hook_configurations`).
		WithArgs(expectedConfiguration.Tag).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"hook_definition_id",
			"tag",
			"url",
			"client_secret",
			"client_rsa_private_key",
			"created_at",
		}).AddRow(
			expectedConfiguration.ID,
			expectedConfiguration.HookDefinitionID,
			expectedConfiguration.Tag,
			expectedConfiguration.URL,
			expectedConfiguration.ClientSecret,
			expectedConfiguration.ClientRSAPrivateKey,
			expectedConfiguration.CreatedAt,
		))

	config, err := persister.FindHookConfigurationsByTag(context.Background(), expectedConfiguration.Tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config == nil {
		t.Fatalf("expected configuration to be non-nil, got nil")
	}
}

func TestSqlPersister_FindHookConfigurations(t *testing.T) {
	persister, mock, close := mustCreateTestPersister(t)
	defer close()

	expectedConfiguration := &HookConfiguration{
		ID:                  "config-id",
		HookDefinitionID:    "hook-definition-id",
		Tag:                 "example-tag",
		URL:                 "http://example.com/config",
		ClientSecret:        nil,
		ClientRSAPrivateKey: nil,
		CreatedAt:           time.Now(),
	}

	mock.ExpectQuery(`SELECT (.+) FROM hook_configurations`).
		WithArgs().
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"hook_definition_id",
			"tag",
			"url",
			"client_secret",
			"client_rsa_private_key",
			"created_at",
		}).AddRow(
			expectedConfiguration.ID,
			expectedConfiguration.HookDefinitionID,
			expectedConfiguration.Tag,
			expectedConfiguration.URL,
			expectedConfiguration.ClientSecret,
			expectedConfiguration.ClientRSAPrivateKey,
			expectedConfiguration.CreatedAt,
		))

	config, err := persister.FindHookConfigurations(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config == nil {
		t.Fatalf("expected configuration to be non-nil, got nil")
	}
}

func TestSqlPersister_FindHookConfiguration_NoRows(t *testing.T) {
	persister, mock, close := mustCreateTestPersister(t)
	defer close()

	mock.ExpectQuery(`SELECT (.+) FROM hook_configurations`).
		WithArgs("foo", "tag").
		WillReturnError(sql.ErrNoRows)

	_, err := persister.FindHookConfiguration(context.Background(), "foo", HookConfigurationTag("tag"))
	if err != ErrNotFound {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSqlPersister_WriteHookConfiguration(t *testing.T) {
	persister, mock, close := mustCreateTestPersister(t)
	defer close()

	now := time.Now().UTC()
	configuration := &HookConfiguration{
		ID:                  "config-id",
		HookDefinitionID:    "hook-definition-id",
		Tag:                 "example-tag",
		URL:                 "http://example.com/config",
		ClientSecret:        nil,
		ClientRSAPrivateKey: nil,
		CreatedAt:           now,
	}

	mock.ExpectExec(`INSERT INTO hook_configurations`).
		WithArgs(
			configuration.ID,
			configuration.HookDefinitionID,
			configuration.URL,
			configuration.Tag,
			configuration.ClientSecret,
			configuration.ClientRSAPrivateKey,
			configuration.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := persister.WriteHookConfiguration(context.Background(), configuration)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSqlPersister_FindHookDefinitionByID(t *testing.T) {
	persister, mock, close := mustCreateTestPersister(t)
	defer close()

	expectedDefinition := &HookDefinition{
		ID:                "hook-definition-id",
		Description:       "Example Hook Definition",
		PayloadScheme:     json.RawMessage(`{"type":"object","properties":{"key":{"type":"string"}}}`),
		HttpRequestMethod: POST,
		TotalAttempts:     5,
	}

	mock.ExpectQuery(`SELECT (.+) FROM hook_definitions`).
		WithArgs(expectedDefinition.ID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"name",
			"description",
			"payload_scheme",
			"http_request_method",
			"total_attempts",
		}).AddRow(
			expectedDefinition.ID,
			expectedDefinition.Name,
			expectedDefinition.Description,
			expectedDefinition.PayloadScheme,
			expectedDefinition.HttpRequestMethod,
			expectedDefinition.TotalAttempts,
		))

	definition, err := persister.FindHookDefinitionByID(context.Background(), expectedDefinition.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if definition == nil {
		t.Fatalf("expected definition to be non-nil, got nil")
	}
}

func TestSqlPersister_FindHookDefinitions(t *testing.T) {
	persister, mock, close := mustCreateTestPersister(t)
	defer close()

	expectedDefinition := &HookDefinition{
		ID:                "hook-definition-id",
		Description:       "Example Hook Definition",
		PayloadScheme:     json.RawMessage(`{"type":"object","properties":{"key":{"type":"string"}}}`),
		HttpRequestMethod: POST,
		TotalAttempts:     5,
	}

	mock.ExpectQuery(`SELECT (.+) FROM hook_definitions`).
		WithArgs().
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"name",
			"description",
			"payload_scheme",
			"http_request_method",
			"total_attempts",
		}).AddRow(
			expectedDefinition.ID,
			expectedDefinition.Name,
			expectedDefinition.Description,
			expectedDefinition.PayloadScheme,
			expectedDefinition.HttpRequestMethod,
			expectedDefinition.TotalAttempts,
		))

	definition, err := persister.FindHookDefinitions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if definition == nil {
		t.Fatalf("expected definition to be non-nil, got nil")
	}
}

func TestSqlPersister_WriteHookDefinitions(t *testing.T) {
	persister, mock, close := mustCreateTestPersister(t)
	defer close()

	firstDefinition := &HookDefinition{
		ID:                "hook-definition-id",
		Description:       "Example Hook Definition",
		PayloadScheme:     json.RawMessage(`{"type":"object","properties":{"key":{"type":"string"}}}`),
		HttpRequestMethod: POST,
		TotalAttempts:     5,
	}
	secondDefinition := &HookDefinition{
		ID:                "hook-definition-id",
		Description:       "Example Hook Definition",
		PayloadScheme:     json.RawMessage(`{"type":"object","properties":{"key":{"type":"string"}}}`),
		HttpRequestMethod: POST,
		TotalAttempts:     5,
	}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO hook_definitions`).
		WithArgs(
			firstDefinition.ID,
			firstDefinition.Name,
			firstDefinition.Description,
			firstDefinition.PayloadScheme,
			firstDefinition.HttpRequestMethod,
			firstDefinition.TotalAttempts).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(`INSERT INTO hook_definitions`).
		WithArgs(
			secondDefinition.ID,
			secondDefinition.Name,
			secondDefinition.Description,
			secondDefinition.PayloadScheme,
			secondDefinition.HttpRequestMethod,
			secondDefinition.TotalAttempts).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := persister.WriteHookDefinitions(context.Background(), firstDefinition, secondDefinition)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
