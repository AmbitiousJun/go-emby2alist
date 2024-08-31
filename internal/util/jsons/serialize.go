package jsons

import (
	"fmt"
	"strings"
)

// Struct 将 item 转换为结构体对象
func (i *Item) Struct() interface{} {
	switch i.jType {
	case JsonTypeVal:
		return i.val
	case JsonTypeObj:
		m := make(map[string]interface{})
		for key, value := range i.obj {
			m[key] = value.Struct()
		}
		return m
	case JsonTypeArr:
		a := make([]interface{}, 0)
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
		switch i.val.(type) {
		case string:
			return fmt.Sprintf(`"%v"`, i.val)
		default:
			return fmt.Sprintf("%v", i.val)
		}
	case JsonTypeObj:
		sb := strings.Builder{}
		sb.WriteString("{")
		cur, tot := 0, len(i.obj)
		for key, value := range i.obj {
			sb.WriteString(fmt.Sprintf(`"%s":%s`, key, value.String()))
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
