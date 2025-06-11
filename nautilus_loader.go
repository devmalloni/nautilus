package nautilus

import (
	"context"
	"encoding/json"
	"os"

	"gopkg.in/yaml.v3"
)

type (
	yamlDefinition struct {
		ID                string              `yaml:"id"`
		Name              string              `yaml:"name"`
		Description       string              `yaml:"description"`
		PayloadScheme     string              `yaml:"payload_scheme"`
		HttpRequestMethod HttpRequestMethod   `yaml:"http_request_method"`
		TotalAttempts     int                 `yaml:"total_attempts"`
		Configurations    []yamlConfiguration `yaml:"configurations"`
	}
	yamlConfiguration struct {
		ID                  string               `yaml:"id"`
		Tag                 HookConfigurationTag `yaml:"tag"`
		URL                 string               `yaml:"url"`
		ClientSecret        *string              `yaml:"client_secret"`
		ClientRSAPrivateKey *string              `yaml:"client_rsa_private_key"`
	}
	nautilusYamlConfig struct {
		Definitions []*yamlDefinition `yaml:"definitions"`
	}
)

func (p *Nautilus) LoadFromYamlBytes(ctx context.Context, data []byte) error {
	config := nautilusYamlConfig{}

	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return err
	}
	var definitions []*HookDefinition
	var configs []*HookConfiguration
	for _, def := range config.Definitions {
		definition := &HookDefinition{
			ID:                def.ID,
			Name:              def.Name,
			Description:       def.Description,
			PayloadScheme:     json.RawMessage(def.PayloadScheme),
			HttpRequestMethod: def.HttpRequestMethod,
			TotalAttempts:     def.TotalAttempts,
		}
		if def.Configurations != nil {
			for _, conf := range def.Configurations {
				configuration := &HookConfiguration{
					ID:                  conf.ID,
					Tag:                 conf.Tag,
					URL:                 conf.URL,
					ClientSecret:        conf.ClientSecret,
					ClientRSAPrivateKey: conf.ClientRSAPrivateKey,
				}
				configs = append(configs, configuration)
			}
		}
		definitions = append(definitions, definition)
	}

	err = p.RegisterDefinitions(ctx, definitions...)
	if err != nil {
		return err
	}

	err = p.RegisterConfigurations(ctx, configs...)
	if err != nil {
		return err
	}

	return nil
}

func (p *Nautilus) LoadFromYamlString(ctx context.Context, yamlString string) error {
	return p.LoadFromYamlBytes(ctx, []byte(yamlString))
}

func (p *Nautilus) LoadFromYamlFile(ctx context.Context, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return p.LoadFromYamlBytes(ctx, data)
}
