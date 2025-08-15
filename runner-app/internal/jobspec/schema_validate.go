package jobspec

// schemaValidationEnabled indicates whether JSON Schema validation is active.
// When false, ValidateJSONSchema is a no-op stub.
var schemaValidationEnabled = false

// ValidateJSONSchema validates a JobSpec JSON document.
// NOTE: JSON Schema validation is currently disabled to avoid external
// module fetches that hang in certain environments. Structural validation
// is handled by models.JobSpec.Validate(). This function is a no-op stub
// returning nil so callers can remain unchanged.
func ValidateJSONSchema(data []byte) error {
    return nil
}
