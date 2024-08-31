package model

// HttpRes 通用 http 请求结果
type HttpRes[T any] struct {
	Code int
	Data T
	Msg  string
}
