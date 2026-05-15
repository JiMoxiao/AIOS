package evaluator

import (
	"encoding/json"
	"errors"
)

func ValidateJSONAgainstSchema(output string, schema any) error {
	switch s := schema.(type) {
	case bool:
		if !s {
			return errors.New("schema_false")
		}
		var v any
		if err := json.Unmarshal([]byte(output), &v); err != nil {
			return err
		}
		return nil
	default:
		var v any
		if err := json.Unmarshal([]byte(output), &v); err != nil {
			return err
		}
		return nil
	}
}

