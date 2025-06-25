package jsons

import (
	"encoding/json"
	"strings"
)

// Struct 将 item 转换为结构体对象
func (i *Item) Struct() any {
	switch i.jType {
	case JsonTypeVal:
		return i.val
	case JsonTypeObj:
		m := make(map[string]any)
		for key, value := range i.obj {
			m[key] = value.Struct()
		}
		return m
	case JsonTypeArr:
		a := make([]any, len(i.arr))
		for idx, value := range i.arr {
			a[idx] = value.Struct()
		}
		return a
	default:
		return "null"
	}
}

// String 将 item 转换为 json 字符串
func (i *Item) String() string {
	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(i.Struct())
	if err != nil {
		return "null"
	}
	return strings.TrimSpace(buf.String())
}
