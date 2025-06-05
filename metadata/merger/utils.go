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
	output := make([]any, 0, len(input))
	for i := range input {
		item := input[i]
		switch typedItem := item.(type) {
		case map[string]any:
			output = append(output, DeepCopyMap(typedItem))

		case []any:
			output = append(output, DeepCopySlice(typedItem))

		default:
			output = append(output, typedItem)
		}
	}

	return output
}
