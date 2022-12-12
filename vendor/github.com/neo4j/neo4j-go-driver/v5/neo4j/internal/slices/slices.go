package slices

func Contains[T comparable](items []T, expectedItem T) bool {
	for _, item := range items {
		if item == expectedItem {
			return true
		}
	}
	return false
}
