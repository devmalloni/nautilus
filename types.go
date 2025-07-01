package nautilus

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/devmalloni/nautilus/x"
)

const (
	ClientSecretHeader    = "X-Client-Secret"
	ClientSignatureHeader = "X-Client-Signature"
)

const (
	GET    HttpRequestMethod = "GET"
	POST   HttpRequestMethod = "POST"
	PUT    HttpRequestMethod = "PUT"
	DELETE HttpRequestMethod = "DELETE"
)

const (
	HookScheduleStatusScheduled HookScheduleStatus = "scheduled"
	HookScheduleStatusExecuted  HookScheduleStatus = "executed"
	HookScheduleStatusFailed    HookScheduleStatus = "failed"
)

type (
	JSchemaValidator interface {
		Validate(schema, payload json.RawMessage) error
	}

	HttpRequestMethod    string
	HookConfigurationTag string
	HookScheduleStatus   string

	HookDefinition struct {
		ID string `json:"id,omitempty" yaml:"id" db:"id"`
		/*
		* Name of hook definition
		*
		* e.g entity_created
		 */
		Name string `json:"name,omitempty" yaml:"name" db:"name"`
		/*
		* Hook full description
		*
		* e.g triggered when an entity is created
		 */
		Description string `json:"description,omitempty" yaml:"description" db:"description"`
		/*
		* Specifies the json scheme that the request payload must attend
		*
		* e.g { "properties": { "created_at": { "type": "string" } } }
		 */
		PayloadScheme json.RawMessage `json:"payload_scheme,omitempty" yaml:"payload_scheme" db:"payload_scheme"`
		/*
		* Method that will be used to make the request
		*
		* e.g GET
		 */
		HttpRequestMethod HttpRequestMethod `json:"http_request_method,omitempty" yaml:"http_request_method" db:"http_request_method"`
		/*
		* Max attempts system will try to deliver this message to a configuration
		 */
		TotalAttempts int `json:"total_attempts,omitempty" yaml:"total_attempts" db:"total_attempts"`

		/*
		* Specifies if payload must be sent in raw or if it will be included in execution metadata (e.g. sent at, definition id and unique execution id)
		 */
		HideExecutionMetadata bool `json:"hide_execution_metadata,omitempty" yaml:"hide_execution_metadata" db:"hide_execution_metadata"`
	}

	HookConfiguration struct {
		ID               string               `json:"id,omitempty" yaml:"id" db:"id"`
		HookDefinitionID string               `json:"hook_definition_id,omitempty" yaml:"hook_definition_id" db:"hook_definition_id"`
		Tag              HookConfigurationTag `json:"tag,omitempty" yaml:"tag" db:"tag"`
		URL              string               `json:"url,omitempty" yaml:"url" db:"url"`

		ClientSecret        *string `json:"client_secret,omitempty" yaml:"client_secret" db:"client_secret"`
		ClientRSAPrivateKey *string `json:"-,omitempty" yaml:"client_rsa_private_key" db:"client_rsa_private_key"`

		CreatedAt time.Time `json:"created_at,omitempty" yaml:"created_at" db:"created_at"`

		HookDefinition *HookDefinition `json:"hook_definition,omitempty"`
	}

	HookSchedule struct {
		ID                  string             `json:"id,omitempty" db:"id"`
		HookConfigurationID string             `json:"hook_configuration_id,omitempty" db:"hook_configuration_id"`
		HttpRequestMethod   HttpRequestMethod  `json:"http_request_method,omitempty" db:"http_request_method"`
		URL                 string             `json:"url,omitempty" db:"url"`
		Payload             json.RawMessage    `json:"payload,omitempty" db:"payload"`
		Status              HookScheduleStatus `json:"status,omitempty" db:"status"`

		MaxAttempt            int  `json:"max_attempt,omitempty" db:"max_attempt"`
		CurrentAttempt        int  `json:"current_attempt,omitempty" db:"current_attempt"`
		HideExecutionMetadata bool `json:"hide_execution_metadata,omitempty" db:"hide_execution_metadata"`

		CreatedAt time.Time  `json:"created_at,omitempty" db:"created_at"`
		UpdatedAt *time.Time `json:"updated_at,omitempty" db:"updated_at"`

		HookConfiguration *HookConfiguration `json:"hook_configuration,omitempty"`
	}

	HookExecution struct {
		ID              string          `json:"id,omitempty" db:"id"`
		HookScheduleID  string          `json:"hook_schedule_id,omitempty" db:"hook_schedule_id"`
		ResponseStatus  int             `json:"response_status,omitempty" db:"response_status"`
		ResponsePayload json.RawMessage `json:"response_payload,omitempty" db:"response_payload"`
		CreatedAt       time.Time       `json:"created_at,omitempty" db:"created_at"`
	}

	HookExecutionData struct {
		ID               string          `json:"id,omitempty"`
		SentAt           time.Time       `json:"sent_at,omitempty"`
		HookDefinitionID string          `json:"hook_definition_id,omitempty"`
		Data             json.RawMessage `json:"data,omitempty"`
	}
)

func (p *HookDefinition) CreateConfiguration(id, url string,
	tag HookConfigurationTag,
	clientSecret *string) (*HookConfiguration, error) {
	hc := &HookConfiguration{
		ID:               id,
		HookDefinitionID: p.ID,
		URL:              url,
		Tag:              tag,
		ClientSecret:     clientSecret,
		CreatedAt:        time.Now().UTC(),
		HookDefinition:   p,
	}

	if err := hc.IsValid(); err != nil {
		return nil, err
	}

	return hc, nil
}

func (p *HookDefinition) IsValid() error {
	if p.ID == "" {
		return errors.New("id is required")
	}

	switch p.HttpRequestMethod {
	case GET, POST, PUT, DELETE:
	default:
		return errors.New("http request method is not valid")
	}

	if p.TotalAttempts <= 0 {
		return errors.New("total attempts must be higher than 0")
	}

	return nil
}

func (p *HookConfiguration) IsValid() error {
	_, err := url.ParseRequestURI(p.URL)
	if err != nil {
		return err
	}

	if p.Tag == "" {
		return errors.New("tag is required")
	}

	if p.HookDefinitionID == "" {
		return errors.New("hook definition id must be set")
	}

	if p.HookDefinition == nil {
		return errors.New("hook definition is not set")
	}

	return nil
}

func (p *HookConfiguration) Schedule(id string, payload json.RawMessage, validator JSchemaValidator) (*HookSchedule, error) {
	if p.HookDefinition.PayloadScheme != nil && validator != nil {
		if err := validator.Validate(p.HookDefinition.PayloadScheme, payload); err != nil {
			return nil, err
		}
	}

	s := &HookSchedule{
		ID:                    id,
		HookConfigurationID:   p.ID,
		URL:                   p.URL,
		Payload:               payload,
		HttpRequestMethod:     p.HookDefinition.HttpRequestMethod,
		Status:                HookScheduleStatusScheduled,
		MaxAttempt:            p.HookDefinition.TotalAttempts,
		HideExecutionMetadata: p.HookDefinition.HideExecutionMetadata,
		HookConfiguration:     p,
		CurrentAttempt:        0,
		CreatedAt:             time.Now().UTC(),
	}

	if err := s.IsValid(); err != nil {
		return nil, err
	}

	return s, nil
}

func (p *HookConfiguration) GeneratePrivateKey(override bool) error {
	if p.ClientRSAPrivateKey != nil && !override {
		return errors.New("private key already set")
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Validate the private key
	err = privateKey.Validate()
	if err != nil {
		return err
	}

	// Marshal the private key into PKCS#1 ASN.1 DER encoded form
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	// Create a PEM block
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
	}

	// Encode to PEM format and output
	pemFile := pem.EncodeToMemory(privBlock)
	pemFileStr := string(pemFile)
	p.ClientRSAPrivateKey = &pemFileStr

	return nil
}

func (p *HookSchedule) IsValid() error {

	if p.HookConfigurationID == "" {
		return errors.New("hook configuration id is required")
	}

	if p.HookConfiguration == nil {
		return errors.New("hook configuration is not set")
	}
	url, err := url.Parse(p.URL)
	if err != nil {
		return err
	}

	if url.Scheme != "http" && url.Scheme != "https" {
		return errors.New("url scheme must be http or https")
	}

	if p.MaxAttempt <= 0 {
		return errors.New("max attempt must be higher than 0")
	}

	return nil
}

func (p *HookSchedule) Execute(ctx context.Context, executionID string, client *http.Client) (*HookExecution, error) {
	e := &HookExecution{
		ID:              executionID,
		HookScheduleID:  p.ID,
		ResponseStatus:  0,
		ResponsePayload: nil,
		CreatedAt:       time.Now().UTC(),
	}

	b, err := p.createExecutionData(e)
	if err != nil {
		return nil, err
	}

	buff := bytes.NewBuffer(b)
	req, err := http.NewRequestWithContext(ctx,
		string(p.HttpRequestMethod),
		p.URL,
		buff)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.HookConfiguration.ClientSecret != nil {
		req.Header.Set(ClientSecretHeader, *p.HookConfiguration.ClientSecret)
	}

	if p.HookConfiguration.ClientRSAPrivateKey != nil {
		rsaSignature, err := signBody(b, *p.HookConfiguration.ClientRSAPrivateKey)
		if err != nil {
			return nil, err
		}
		req.Header.Set(ClientSignatureHeader, rsaSignature)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	e.ResponseStatus = resp.StatusCode
	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		e.ResponsePayload = json.RawMessage(fmt.Sprintf("unable to retrieve json response: %v", err))
	} else {
		e.ResponsePayload = json.RawMessage(responseBytes)
	}

	p.CurrentAttempt++
	if resp.StatusCode == http.StatusOK {
		p.Status = HookScheduleStatusExecuted
	} else if p.CurrentAttempt > p.MaxAttempt {
		p.Status = HookScheduleStatusFailed
	}
	p.UpdatedAt = x.NilTime(time.Now().UTC())

	return e, nil
}

func (p *HookSchedule) createExecutionData(e *HookExecution) ([]byte, error) {
	var requestData any
	if p.HideExecutionMetadata {
		requestData = p.Payload
	} else {
		requestData = HookExecutionData{
			ID:               e.HookScheduleID,
			SentAt:           time.Now().UTC(),
			HookDefinitionID: p.HookConfiguration.HookDefinitionID,
			Data:             p.Payload,
		}
	}

	b, err := json.Marshal(&requestData)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// SignBody signs the body using SHA256 and RSA.
func signBody(body []byte, privateKey string) (string, error) {
	rsaPrivateKey, err := rsaPemToPrivateKey(privateKey)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(body)
	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaPrivateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

func verifySignature(body []byte, signature string, publicKey *rsa.PublicKey) error {
	if publicKey == nil {
		return errors.New("public key is nil")
	}

	hash := sha256.Sum256(body)
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return err
	}

	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], sigBytes)
	if err != nil {
		return err
	}

	return nil
}

func rsaPemToPrivateKey(pemstr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemstr))
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")

	}

	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		fmt.Println("failed to parse RSA private key:", err)
		return nil, err
	}

	return privKey, nil
}

func (p *HookConfiguration) PublicKey() (*rsa.PublicKey, error) {
	privKey, err := rsaPemToPrivateKey(*p.ClientRSAPrivateKey)
	if err != nil {
		return nil, err
	}

	return &privKey.PublicKey, nil
}
