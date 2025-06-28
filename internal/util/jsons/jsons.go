package jsons

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/parallels"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/strs"
)

// NewEmptyObj 初始化一个对象类型的 json 数据
func NewEmptyObj() *Item {
	return &Item{obj: make(map[string]*Item), jType: JsonTypeObj}
}

// NewEmptyArr 初始化一个数组类型的 json 数据
func NewEmptyArr() *Item {
	return &Item{arr: make([]*Item, 0), jType: JsonTypeArr}
}

// FromObject 根据对象初始化 json 数据
func FromObject(obj any) *Item {
	if obj == nil {
		return FromValue(nil)
	}

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct && v.Kind() != reflect.Map {
		return FromValue(obj)
	}

	item := NewEmptyObj()
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			fieldVal := v.Field(i)
			fieldType := v.Type().Field(i)
			if !fieldVal.CanInterface() {
				continue
			}
			item.Put(fieldType.Name, FromValue(fieldVal.Interface()))
		}
	}

	if v.Kind() == reflect.Map {
		if v.Type().Key() != reflect.TypeOf("") {
			panic("不支持的 map 类型")
		}
		for _, key := range v.MapKeys() {
			item.Put(key.Interface().(string), FromValue(v.MapIndex(key).Interface()))
		}
	}
	return item
}

// FromArray 根据数组初始化 json 数据
func FromArray(arr any) *Item {
	if arr == nil {
		return FromValue(nil)
	}

	v := reflect.ValueOf(arr)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Array && v.Kind() != reflect.Slice {
		return FromValue(arr)
	}

	item := NewEmptyArr()
	for i := 0; i < v.Len(); i++ {
		field := v.Index(i)
		if !field.CanInterface() {
			continue
		}
		item.Append(FromValue(field.Interface()))
	}
	return item
}

// FromValue 根据指定普通值初始化 json 数据, 如果是数组或对象类型也会自动转化
func FromValue(val any) *Item {
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
		return FromObject(val)
	case reflect.Array, reflect.Slice:
		return FromArray(val)
	default:
		log.Panicf("无效的数据类型, kind: %v, name: %v", t.Kind(), t.Name())
		return nil
	}
}

// New 从 json 字符串中初始化成 item 对象
func New(rawJson string) (i *Item, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("内部转换异常: %v", rec)
		}
	}()

	if strs.AnyEmpty(rawJson) || rawJson == "null" {
		return FromValue(nil), nil
	}

	if strings.HasPrefix(rawJson, "{") {
		var data map[string]json.RawMessage
		if err := json.Unmarshal([]byte(rawJson), &data); err != nil {
			return nil, err
		}
		return makeObject(data)
	}

	if strings.HasPrefix(rawJson, "[") {
		var data []json.RawMessage
		if err := json.Unmarshal([]byte(rawJson), &data); err != nil {
			return nil, err
		}
		return makeArray(data)
	}

	// 尝试转换成基础类型
	var v any
	if err := json.Unmarshal([]byte(rawJson), &v); err != nil {
		return nil, fmt.Errorf("不支持的字符串: %s", rawJson)
	}
	return FromValue(v), nil
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

// makeObject 根据原始对象数据构造对象 Item
func makeObject(rawData map[string]json.RawMessage) (*Item, error) {
	item := NewEmptyObj()
	if len(rawData) == 0 {
		return item, nil
	}

	// 并行构造子项
	type result struct {
		key  string
		item *Item
		err  error
	}
	results := make(chan result, runtime.NumCPU()*2)
	wg := sync.WaitGroup{}

	for key, value := range rawData {
		wg.Add(1)
		go func(key string, value json.RawMessage) {
			defer wg.Done()
			subI, err := New(string(value))
			results <- result{key: key, item: subI, err: err}
		}(key, value)
	}

	// 异步关闭结果收集通道
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	for r := range results {
		if r.err != nil {
			return nil, r.err
		}
		item.Put(r.key, r.item)
	}
	return item, nil
}

// makeArray 根据原始数组数据构造数组 Item
func makeArray(rawData []json.RawMessage) (*Item, error) {
	item := NewEmptyArr()
	if len(rawData) == 0 {
		return item, nil
	}

	// 分块处理
	ranges := parallels.SliceChunk(len(rawData))

	// 并行构造子项
	type result struct {
		idx  int
		item *Item
		err  error
	}
	results := make(chan result, runtime.NumCPU()*2)
	wg := sync.WaitGroup{}

	for _, r := range ranges {
		wg.Add(1)
		go func(r parallels.Range) {
			defer wg.Done()
			for i := range r.End - r.Start {
				subI, err := New(string(rawData[i+r.Start]))
				results <- result{idx: i + r.Start, item: subI, err: err}
			}
		}(r)
	}

	// 异步关闭结果收集通道
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	for r := range results {
		if r.err != nil {
			return nil, r.err
		}
		item.PutIdx(r.idx, r.item)
	}
	return item, nil
}
