package compatibility

func ToSet[K comparable](src []K) map[K]struct{} {
	var result = make(map[K]struct{})
	for _, v := range src {
		result[v] = struct{}{}
	}
	return result
}
