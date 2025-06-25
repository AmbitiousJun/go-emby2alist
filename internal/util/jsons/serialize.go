package jsons

import (
	"encoding/json"
	"fmt"
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
		a := make([]any, 0)
		for _, value := range i.arr {
			a = append(a, value.Struct())
		}
		return a
	default:
		return "Error jType"
	}
}

// String 将 item 转换为 json 字符串
func (i *Item) String() string {
	switch i.jType {
	case JsonTypeVal:
		if i.val == nil {
			return "null"
		}

		bytes, _ := json.Marshal(i.val)
		str := string(bytes)
		return str
	case JsonTypeObj:
		sb := strings.Builder{}
		sb.WriteString("{")
		cur, tot := 0, len(i.obj)
		for key, value := range i.obj {
			kb, _ := json.Marshal(key)
			sb.WriteString(fmt.Sprintf(`%s:%s`, string(kb), value.String()))
			cur++
			if cur != tot {
				sb.WriteString(",")
			}
		}
		sb.WriteString("}")
		return sb.String()
	case JsonTypeArr:
		sb := strings.Builder{}
		sb.WriteString("[")
		for idx, value := range i.arr {
			sb.WriteString(value.String())
			if idx < len(i.arr)-1 {
				sb.WriteString(",")
			}
		}
		sb.WriteString("]")
		return sb.String()
	default:
		return "Error jType"
	}
}
