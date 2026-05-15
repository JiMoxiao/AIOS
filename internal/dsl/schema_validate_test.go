package dsl

import "testing"

func TestValidateWorkflowSpec_MissingRequiredField(t *testing.T) {
	spec := []byte(`{"spec_version":"1.0"}`)
	if err := ValidateWorkflowSpecJSON(spec); err == nil {
		t.Fatalf("expected validation error")
	}
}

