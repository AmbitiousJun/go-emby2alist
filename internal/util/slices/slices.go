package slices

// Copy 拷贝切片
func Copy[T any](src []T) []T {
	if len(src) == 0 {
		return []T{}
	}
	return append(([]T)(nil), src...)
}
