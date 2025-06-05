package merger

func DeepCopyMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}

	output := make(map[string]any)
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
