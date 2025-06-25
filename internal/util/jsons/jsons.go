package jsons

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"

	"github.com/AmbitiousJun/go-emby2openlist/internal/util/strs"
)

// NewEmptyObj 初始化一个对象类型的 json 数据
func NewEmptyObj() *Item {
	return &Item{obj: make(map[string]*Item), jType: JsonTypeObj}
}

// NewEmptyArr 初始化一个数组类型的 json 数据
func NewEmptyArr() *Item {
	return &Item{arr: make([]*Item, 0), jType: JsonTypeArr}
}

// NewByObj 根据对象初始化 json 数据
func NewByObj(obj any) *Item {
	if obj == nil {
		return NewByVal(nil)
	}

	if item, ok := obj.(*Item); ok {
		return item
	}

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct && v.Kind() != reflect.Map {
		return NewByVal(obj)
	}

	item := NewEmptyObj()
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			fieldVal := v.Field(i)
			fieldType := v.Type().Field(i)
			item.Put(fieldType.Name, NewByVal(fieldVal.Interface()))
		}
	}

	if v.Kind() == reflect.Map {
		if v.Type().Key() != reflect.TypeOf("") {
			panic("不支持的 map 类型")
		}
		for _, key := range v.MapKeys() {
			item.Put(key.Interface().(string), NewByVal(v.MapIndex(key).Interface()))
		}
	}
	return item
}

// NewByArr 根据数组初始化 json 数据
func NewByArr(arr any) *Item {
	if arr == nil {
		return NewByVal(nil)
	}

	if item, ok := arr.(*Item); ok {
		return item
	}

	v := reflect.ValueOf(arr)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Array && v.Kind() != reflect.Slice {
		return NewByVal(arr)
	}

	item := NewEmptyArr()
	for i := 0; i < v.Len(); i++ {
		item.Append(NewByVal(v.Index(i).Interface()))
	}
	return item
}

// NewByVal 根据指定普通值初始化 json 数据, 如果是数组或对象类型也会自动转化
func NewByVal(val any) *Item {
	item := &Item{jType: JsonTypeVal}
	if val == nil {
		return item
	}

	if newVal, ok := val.(*Item); ok {
		return newVal
	}

	t := reflect.TypeOf(val)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Float64, reflect.Int64, reflect.String:
		item.val = val
		return item
	case reflect.Struct, reflect.Map:
		return NewByObj(val)
	case reflect.Array, reflect.Slice:
		return NewByArr(val)
	default:
		log.Panicf("无效的数据类型, kind: %v, name: %v", t.Kind(), t.Name())
		return nil
	}
}

// New 从 json 字符串中初始化成 item 对象
func New(rawJson string) (*Item, error) {
	if strs.AnyEmpty(rawJson) {
		return nil, errors.New("empty raw json")
	}

	if rawJson == "null" {
		return NewByVal(nil), nil
	}

	if strings.HasPrefix(rawJson, "{") {
		var data map[string]json.RawMessage
		if err := json.Unmarshal([]byte(rawJson), &data); err != nil {
			return nil, err
		}

		item := NewEmptyObj()
		for key, value := range data {
			subI, err := New(string(value))
			if err != nil {
				return nil, err
			}
			item.Put(key, subI)
		}
		return item, nil
	}

	if strings.HasPrefix(rawJson, "[") {
		var data []json.RawMessage
		if err := json.Unmarshal([]byte(rawJson), &data); err != nil {
			return nil, err
		}

		item := NewEmptyArr()
		for _, value := range data {
			subI, err := New(string(value))
			if err != nil {
				return nil, err
			}
			item.Append(subI)
		}
		return item, nil
	}

	// 尝试转换成基础类型
	var s string
	if err := json.Unmarshal([]byte(rawJson), &s); err == nil {
		return NewByVal(s), nil
	}
	var b bool
	if err := json.Unmarshal([]byte(rawJson), &b); err == nil {
		return NewByVal(b), nil
	}
	var i int
	if err := json.Unmarshal([]byte(rawJson), &i); err == nil {
		return NewByVal(i), nil
	}
	var i64 int64
	if err := json.Unmarshal([]byte(rawJson), &i64); err == nil {
		return NewByVal(i64), nil
	}
	var f float64
	if err := json.Unmarshal([]byte(rawJson), &f); err == nil {
		return NewByVal(f), nil
	}

	return nil, fmt.Errorf("不支持的字符串: %s", rawJson)
}

// Read 从流中读取 JSON 数据并转换为对象
func Read(reader io.Reader) (*Item, error) {
	if reader == nil {
		return nil, errors.New("reader 为空")
	}

	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("读取 reader 数据失败: %v", err)
	}
	return New(string(bytes))
}
