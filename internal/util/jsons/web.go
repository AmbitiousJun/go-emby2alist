package jsons

import (
	"net/http"
	"strconv"
)

// WebStringWriter 能够进行 web 响应的字符串写处理器
type WebStringWriter interface {

	// String 写入字符串响应
	String(code int, format string, values ...any)

	// Header 设置响应头
	Header(key, value string)
}

// OkResp 返回 json 响应, 状态码 200
func OkResp(w WebStringWriter, data *Item) {
	Resp(w, http.StatusOK, data)
}

// Resp 返回 json 响应
func Resp(w WebStringWriter, code int, data *Item) {
	if data == nil {
		data = NewEmptyObj()
	}
	str := data.String()
	w.Header("Content-Length", strconv.Itoa(len(str)))
	w.String(code, str)
}
