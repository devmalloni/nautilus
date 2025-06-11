package nautilus

import (
	"encoding/json"
	"errors"

	"github.com/xeipuuv/gojsonschema"
)

type StandardJsonSchemaValidator struct{}

func NewStandardJsonSchemaValidator() *StandardJsonSchemaValidator {
	return &StandardJsonSchemaValidator{}
}

func (p *StandardJsonSchemaValidator) Validate(schema, data json.RawMessage) error {
	schemaLoader := gojsonschema.NewStringLoader(string(schema))
	documentLoader := gojsonschema.NewStringLoader(string(data))

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		var errs []error
		for _, desc := range result.Errors() {
			errs = append(errs, errors.New(desc.String()))
		}

		return errors.Join(errs...)
	}

	return nil
}
