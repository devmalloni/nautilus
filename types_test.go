package nautilus

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHookDefinition_CreateConfiguration(t *testing.T) {
	d := HookDefinition{
		ID: "test-definition",
	}

	_, err := d.CreateConfiguration("config-id", "http://example.com/hook", "test-tag", ID("secret-id"))
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestHookDefinition_IsValid(t *testing.T) {
	d := HookDefinition{
		ID:                "test-definition",
		HttpRequestMethod: POST,
		Name:              "Test Hook",
		Description:       "This is a test hook definition",
		PayloadScheme:     json.RawMessage(`{"type": "object", "properties": {"key": {"type": "string"}}}`),
		TotalAttempts:     3,
	}

	err := d.IsValid()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestHookConfiguration_IsValid(t *testing.T) {
	hc := HookConfiguration{
		ID:               "config-id",
		HookDefinitionID: "test-definition",
		URL:              "http://example.com/hook",
		Tag:              "test-tag",
		ClientSecret:     nil,
		HookDefinition:   &HookDefinition{},
	}

	err := hc.IsValid()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestHookConfiguration_Schedule(t *testing.T) {
	hc := HookConfiguration{
		ID:               "config-id",
		HookDefinitionID: "test-definition",
		URL:              "http://example.com/hook",
		Tag:              "test-tag",
		ClientSecret:     nil,
		HookDefinition: &HookDefinition{
			ID:                "test-definition",
			TotalAttempts:     3,
			HttpRequestMethod: POST,
			PayloadScheme:     json.RawMessage(`{"type": "object", "required": ["key"], "additionalProperties": false, "properties": {"key": {"type": "string"}}}`)},
	}

	hs, err := hc.Schedule("schedule-id",
		json.RawMessage(`{"key": "value"}`),
		NewStandardJsonSchemaValidator())
	if err != nil {
		t.Errorf("expected no error at hc.Schedule, got %v", err)
		return
	}
	err = hs.IsValid()
	if err != nil {
		t.Errorf("expected no error hs.IsValid, got %v", err)
	}

	_, err = hc.Schedule("schedule-id",
		json.RawMessage(`{"not_key": "value"}`),
		NewStandardJsonSchemaValidator())

	if err == nil {
		t.Errorf("expected error at hc.Schedule with invalid payload, got nil")
		return
	}
}

func TestHookConfiguration_GeneratePrivateKey(t *testing.T) {
	hc := HookConfiguration{
		ID:               "config-id",
		HookDefinitionID: "test-definition",
		URL:              "http://example.com/hook",
		Tag:              "test-tag",
		ClientSecret:     nil,
		HookDefinition:   &HookDefinition{},
	}

	err := hc.GeneratePrivateKey(false)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
		return
	}

	if hc.ClientRSAPrivateKey == nil {
		t.Error("expected ClientRSAPrivateKey to be set, got nil")
		return
	}

	// TODO: test rsa
}

func TestHookSchedule_IsValid(t *testing.T) {
	hs := HookSchedule{
		ID:                  "schedule-id",
		HookConfigurationID: "config-id",
		URL:                 "http://example.com/hook",
		Payload:             json.RawMessage(`{"key": "value"}`),
		HookConfiguration:   &HookConfiguration{},
		MaxAttempt:          3,
	}

	err := hs.IsValid()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
		return
	}
}

func TestHookSchedule_HasInvalidURL(t *testing.T) {
	hs := HookSchedule{
		ID:                  "schedule-id",
		HookConfigurationID: "config-id",
		URL:                 "foo",
		Payload:             json.RawMessage(`{"key": "value"}`),
		HookConfiguration:   &HookConfiguration{},
		MaxAttempt:          3,
	}

	err := hs.IsValid()
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
		return
	}
}

func TestHookSchedule_Execute(t *testing.T) {
	wasWebhookCalled := false
	hasClientSecret := false
	clientSecret := "secret-id"

	hc := &HookConfiguration{
		ClientSecret: &clientSecret,
	}

	err := hc.GeneratePrivateKey(false)
	if err != nil {
		t.Errorf("expected no error at hc.GeneratePrivateKey, got %v", err)
		return
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		wasWebhookCalled = true
		hasClientSecret = req.Header.Get(ClientSecretHeader) == "secret-id"
		clientSignature := req.Header.Get(ClientSignatureHeader)

		publicKey, err := hc.PublicKey()
		if err != nil {
			t.Errorf("expected no error getting public key, got %v", err)
			return
		}

		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("expected no error reading request body, got %v", err)
			return
		}
		expectedPayload := struct {
			SentAt *string `json:"sent_at,omitempty"`
		}{}

		err = json.Unmarshal(bodyBytes, &expectedPayload)
		if err != nil {
			t.Error("expected bodyBytes to be unmarshaled")
			return
		}

		if expectedPayload.SentAt == nil {
			t.Error("expected SentAt to be present in the payload, but it was not")
			return
		}

		err = verifySignature(bodyBytes, clientSignature, publicKey)
		if err != nil {
			t.Errorf("expected no error verifying signature, got %v", err)
			return
		}

		res.WriteHeader(200)
	}))
	defer func() { testServer.Close() }()

	hs := HookSchedule{
		ID:                  "schedule-id",
		HookConfigurationID: "config-id",
		URL:                 testServer.URL,
		Payload:             json.RawMessage(`{"key": "value"}`),
		MaxAttempt:          3,
		HookConfiguration:   hc,
		HttpRequestMethod:   POST,
		Status:              HookScheduleStatusScheduled,
	}

	_, err = hs.Execute(context.Background(), "execution-id", http.DefaultClient)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
		return
	}

	if hs.Status != HookScheduleStatusExecuted {
		t.Errorf("expected status to be %s, got %s", HookScheduleStatusExecuted, hs.Status)
	}

	if !wasWebhookCalled {
		t.Error("expected webhook to be called, but it was not")
		return
	}

	if !hasClientSecret {
		t.Error("expected client secret to be sent, but it was not")
		return
	}
}

func TestHookSchedule_Execute_HideExecutionMetadata(t *testing.T) {
	wasWebhookCalled := false
	hasClientSecret := false
	clientSecret := "secret-id"

	hc := &HookConfiguration{
		ClientSecret: &clientSecret,
	}

	err := hc.GeneratePrivateKey(false)
	if err != nil {
		t.Errorf("expected no error at hc.GeneratePrivateKey, got %v", err)
		return
	}

	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		wasWebhookCalled = true
		hasClientSecret = req.Header.Get(ClientSecretHeader) == "secret-id"
		clientSignature := req.Header.Get(ClientSignatureHeader)

		publicKey, err := hc.PublicKey()
		if err != nil {
			t.Errorf("expected no error getting public key, got %v", err)
			return
		}

		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("expected no error reading request body, got %v", err)
			return
		}

		expectedPayload := struct {
			SentAt *string `json:"sent_at,omitempty"`
		}{}

		err = json.Unmarshal(bodyBytes, &expectedPayload)
		if err != nil {
			t.Error("expected bodyBytes to be unmarshaled")
			return
		}

		if expectedPayload.SentAt != nil {
			t.Error("expected SentAt to be nil in the payload, but it was not")
			return
		}

		err = verifySignature(bodyBytes, clientSignature, publicKey)
		if err != nil {
			t.Errorf("expected no error verifying signature, got %v", err)
			return
		}

		res.WriteHeader(200)
	}))
	defer func() { testServer.Close() }()

	hs := HookSchedule{
		ID:                    "schedule-id",
		HookConfigurationID:   "config-id",
		URL:                   testServer.URL,
		Payload:               json.RawMessage(`{"key": "value"}`),
		MaxAttempt:            3,
		HookConfiguration:     hc,
		HttpRequestMethod:     POST,
		Status:                HookScheduleStatusScheduled,
		HideExecutionMetadata: true,
	}

	_, err = hs.Execute(context.Background(), "execution-id", http.DefaultClient)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
		return
	}

	if hs.Status != HookScheduleStatusExecuted {
		t.Errorf("expected status to be %s, got %s", HookScheduleStatusExecuted, hs.Status)
	}

	if !wasWebhookCalled {
		t.Error("expected webhook to be called, but it was not")
		return
	}

	if !hasClientSecret {
		t.Error("expected client secret to be sent, but it was not")
		return
	}
}
