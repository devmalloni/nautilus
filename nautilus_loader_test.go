package nautilus

import (
	"context"
	"testing"
)

const yamlConfigurationFileTest = `
definitions:
  - id: on_created
    name: on entity created
    description: Triggered when an entity is created
    payload_scheme: |
      {
        "type": "object",
        "properties": {
          "entity_id": {
            "type": "string",
            "description": "The ID of the created entity"
          }
        }
      }
    http_request_method: "POST"
    total_attempts: 10
configs:
  - id: default
    tag: global
    hook_definition_id: on_created
    url: https://localhost:3333/webhook
    client_secret: your_client_secret_here`

func TestNautilus_LoadFromYamlString(t *testing.T) {
	n := New()
	err := n.LoadFromYamlString(context.Background(), yamlConfigurationFileTest)
	if err != nil {
		t.Fatalf("Failed to load YAML configuration: %v", err)
	}
}
