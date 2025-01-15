package maps

// Keys 返回一个 map 的所有 key 组成的切片
func Keys[K comparable, V any](m map[K]V) []K {
	res := make([]K, 0)
	if m == nil {
		return res
	}

	for k := range m {
		res = append(res, k)
	}

	return res
}
