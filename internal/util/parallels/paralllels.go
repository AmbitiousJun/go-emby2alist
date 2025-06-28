package parallels

import "runtime"

// Range 用于表示左闭右开的区间
type Range struct{ Start, End int }

// SliceChunk 基于 CPU 核心数对切片进行分块, 返回区间切片
func SliceChunk(size int) (ranges []Range) {
	if size <= 0 {
		return
	}

	// 计算分块数
	chunkNum := min(runtime.NumCPU(), size)

	// 根据分块数 判断每块大小 (向上取值)
	chunkSize := (size + chunkNum - 1) / chunkNum

	// 分块
	for i := range chunkNum {
		start := i * chunkSize
		end := min((i+1)*chunkSize, size)
		ranges = append(ranges, Range{Start: start, End: end})
	}
	return
}
