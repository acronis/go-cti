package jsonschema

import (
	"fmt"

	"github.com/acronis/go-stacktrace"
	"github.com/xeipuuv/gojsonschema"
)

func MustCompileSchema(schema string) *gojsonschema.Schema {
	s, err := gojsonschema.NewSchemaLoader().Compile(gojsonschema.NewStringLoader(schema))
	if err != nil {
		panic(fmt.Errorf("compile schema: %w", err))
	}
	return s
}

func ValidatorMessagesAsStackTrace(errResults []gojsonschema.ResultError) *stacktrace.StackTrace {
	st := stacktrace.New("validation failed")
	for i := range errResults {
		errResult := errResults[i]
		_ = st.Append(stacktrace.New(errResult.Description(), stacktrace.WithInfo("context", errResult.Context().String("."))))
	}
	return st
}

func CompileJSONSchemaCTIWithValidation(schema *JSONSchemaCTI) (*gojsonschema.Schema, error) {
	// NewRawLoader() will load provided interface{} directly without Marshal/Unmarshal.
	// It is ok since JSONSchemaCTI already uses json.Number that are required by gojsonschema.
	schemaLoader := gojsonschema.NewRawLoader(schema.Map())
	err := ValidateWrapper(CompiledMetaSchemaDraft07, schemaLoader)
	if err != nil {
		return nil, stacktrace.NewWrapped("failed to validate merged schema", err)
	}
	s, err := gojsonschema.NewSchemaLoader().Compile(schemaLoader)
	if err != nil {
		return nil, fmt.Errorf("compile schema loader: %w", err)
	}
	return s, nil
}

// ValidateWrapper is used by public validation methods to validate the data
func ValidateWrapper(s *gojsonschema.Schema, loader gojsonschema.JSONLoader) error {
	res, err := s.Validate(loader)
	if err != nil {
		return fmt.Errorf("schema validate: %w", err)
	}
	if !res.Valid() {
		return ValidatorMessagesAsStackTrace(res.Errors())
	}
	return nil
}

func DeepCopy(input any) any {
	if input == nil {
		return nil
	}

	switch typedValue := input.(type) {
	case map[string]any:
		return DeepCopyMap(typedValue)

	case []any:
		return DeepCopySlice(typedValue)

	default:
		return typedValue
	}
}

func DeepCopyMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}

	output := make(map[string]any, len(input))
	for key, value := range input {
		switch typedValue := value.(type) {
		case map[string]any:
			output[key] = DeepCopyMap(typedValue)

		case []any:
			output[key] = DeepCopySlice(typedValue)

		default:
			output[key] = typedValue
		}
	}

	return output
}

func DeepCopySlice(input []any) []any {
	output := make([]any, len(input))
	for i := range input {
		item := input[i]
		switch typedItem := item.(type) {
		case map[string]any:
			output[i] = DeepCopyMap(typedItem)

		case []any:
			output[i] = DeepCopySlice(typedItem)

		default:
			output[i] = typedItem
		}
	}

	return output
}
