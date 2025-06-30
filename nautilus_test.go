package nautilus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNautilus_Run(t *testing.T) {
	wasWebhookCalled := false
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		wasWebhookCalled = true

		res.WriteHeader(200)
	}))
	defer func() { testServer.Close() }()

	errCh := make(chan error, 100)
	defer close(errCh)
	go func() {
		for err := range errCh {
			t.Errorf("Error in Nautilus: %v", err)
			return
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	persister := NewInMemoryPersister()
	n := New(
		WithPersister(persister),
		WithJsonSchemaValidator(NewStandardJsonSchemaValidator()),
		WithHttpClient(http.DefaultClient),
		WithWorkersCount(5),
		WithScheduleBufferSize(100),
		WithScheduler(NewPollScheduler(persister,
			WithSkipScheduleInterval(1*time.Second),
			WithRunnerInterval(100*time.Millisecond))),
		WithErrCh(errCh))

	err := n.RegisterDefinitions(ctx, &HookDefinition{
		ID:                "on_created",
		Name:              "on entity created",
		Description:       "Triggered when an entity is created",
		PayloadScheme:     json.RawMessage(`{"type": "object", "properties": {"entity_id": {"type": "string", "description": "The ID of the created entity"}}}`),
		HttpRequestMethod: POST,
		TotalAttempts:     10,
	})
	if err != nil {
		t.Fatalf("Failed to register definitions: %v", err)
	}

	err = n.RegisterConfigurations(ctx, &HookConfiguration{
		ID:                  "default",
		HookDefinitionID:    "on_created",
		URL:                 testServer.URL + "/webhook",
		Tag:                 Global,
		ClientSecret:        nil,
		ClientRSAPrivateKey: nil,
	})

	if err != nil {
		t.Fatalf("Failed to register configurations: %v", err)
	}

	go n.Run(ctx)

	n.MustSchedule(ctx, ID("single_id"), "on_created", Global, json.RawMessage(`{"entity_id": "example"}`))

	<-time.After(1 * time.Second)

	if !wasWebhookCalled {
		t.Fatal("Webhook was not called as expected")
	}
}

func TestNautilus_TrySchedule(t *testing.T) {
	errCh := make(chan error, 100)
	defer close(errCh)

	persister := NewInMemoryPersister()
	n := New(
		WithPersister(persister),
		WithJsonSchemaValidator(NewStandardJsonSchemaValidator()),
		WithHttpClient(http.DefaultClient),
		WithWorkersCount(5),
		WithScheduleBufferSize(100),
		WithScheduler(NewPollScheduler(persister,
			WithSkipScheduleInterval(1*time.Second),
			WithRunnerInterval(100*time.Millisecond))),
		WithErrCh(errCh))

	err := n.RegisterDefinitions(context.Background(), &HookDefinition{
		ID:                "on_created",
		Name:              "on entity created",
		Description:       "Triggered when an entity is created",
		PayloadScheme:     json.RawMessage(`{"type": "object", "properties": {"entity_id": {"type": "string", "description": "The ID of the created entity"}}}`),
		HttpRequestMethod: POST,
		TotalAttempts:     10,
	})
	if err != nil {
		t.Fatalf("Failed to register definitions: %v", err)
	}

	if err != nil {
		t.Fatalf("Failed to register configurations: %v", err)
	}

	err = n.TryScheduleJSON(context.Background(),
		ID("single_id"),
		"on_created",
		Global,
		json.RawMessage(`{"entity_id": "example"}`))

	if err != nil {
		t.Fatalf("Failed to register configurations: %v", err)
	}
}
