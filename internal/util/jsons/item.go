package jsons

import (
	"errors"
	"math/rand"
	"sync"
)

// JsonType json 属性值类型
type JsonType string

const (
	JsonTypeVal JsonType = "val"
	JsonTypeObj JsonType = "obj"
	JsonTypeArr JsonType = "arr"
)

// ErrBreakRange 停止遍历
var ErrBreakRange = errors.New("break arr or obj range")

// Item 表示一个 JSON 数据项
type Item struct {

	// val 普通值: string, bool, int, float64, <null>
	val any

	// obj 对象值
	obj map[string]*Item

	// arr 数组值
	arr []*Item

	// jType 当前数据项类型
	jType JsonType

	// mu 并发控制
	mu sync.Mutex
}

// Type 获取 json 项类型
func (i *Item) Type() JsonType {
	return i.jType
}

// Put obj 设置键值对
func (i *Item) Put(key string, value *Item) {
	if i.jType != JsonTypeObj || value == nil {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.obj[key] = value
}

// Attr 获取对象属性的某个 key 值
func (i *Item) Attr(key string) *TempItem {
	ti := &TempItem{item: i}
	return ti.Attr(key)
}

// RangeObj 遍历对象
func (i *Item) RangeObj(callback func(key string, value *Item) error) error {
	if i.jType != JsonTypeObj {
		return nil
	}
	for k, v := range i.obj {
		ck, cv := k, v
		if err := callback(ck, cv); err == ErrBreakRange {
			return nil
		} else if err != nil {
			return err
		}
	}
	return nil
}

// DelKey 删除对象中的指定键
func (i *Item) DelKey(key string) {
	if i.jType != JsonTypeObj {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	delete(i.obj, key)
}

// Append arr 添加属性
func (i *Item) Append(values ...*Item) {
	if i.jType != JsonTypeArr || len(values) == 0 {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	for _, value := range values {
		if value == nil {
			continue
		}
		i.arr = append(i.arr, value)
	}
}

// ValuesArr 获取数组中的所有值
func (i *Item) ValuesArr() []*Item {
	return i.arr
}

// Idx 获取数组属性的指定索引值
func (i *Item) Idx(index int) *TempItem {
	ti := &TempItem{item: i}
	return ti.Idx(index)
}

// PutIdx 设置数组指定索引的 item
func (i *Item) PutIdx(index int, newItem *Item) {
	if i.jType != JsonTypeArr || newItem == nil {
		return
	}
	if index < 0 || index >= len(i.arr) {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.arr[index] = newItem
}

// RangeArr 遍历数组
func (i *Item) RangeArr(callback func(index int, value *Item) error) error {
	if i.jType != JsonTypeArr {
		return nil
	}
	for idx, v := range i.arr {
		ci, cv := idx, v
		if err := callback(ci, cv); err == ErrBreakRange {
			return nil
		} else if err != nil {
			return err
		}
	}
	return nil
}

// FindIdx 在数组中查找符合条件的属性的索引
//
// 查找不到符合条件的属性时, 返回 -1
func (i *Item) FindIdx(filterFunc func(val *Item) bool) int {
	idx := -1
	if i.jType != JsonTypeArr {
		return idx
	}
	i.RangeArr(func(index int, value *Item) error {
		if filterFunc(value) {
			idx = index
			return ErrBreakRange
		}
		return nil
	})
	return idx
}

// Map 将数组中的元素按照指定规则映射之后返回一个新数组
func (i *Item) Map(mapFunc func(val *Item) any) []any {
	if i.jType != JsonTypeArr {
		return nil
	}
	res := make([]any, 0)
	i.RangeArr(func(_ int, value *Item) error {
		res = append(res, mapFunc(value))
		return nil
	})
	return res
}

// Shuffle 打乱一个数组, 只有这个 item 是数组类型时, 才生效
func (i *Item) Shuffle() {
	if i.jType != JsonTypeArr {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	rand.Shuffle(i.Len(), func(j, k int) {
		i.arr[j], i.arr[k] = i.arr[k], i.arr[j]
	})
}

// DelIdx 删除数组元素
func (i *Item) DelIdx(index int) {
	if i.jType != JsonTypeArr {
		return
	}
	if index < 0 || index >= len(i.arr) {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.arr = append(i.arr[:index], i.arr[index+1:]...)
}

// Len 返回子项个数
//
//	如果 Type 为 obj 和 arr, 返回子项个数
//	如果 Type 为 val, 返回 0
func (i *Item) Len() int {
	switch i.jType {
	case JsonTypeObj:
		return len(i.obj)
	case JsonTypeArr:
		return len(i.arr)
	default:
		return 0
	}
}

// Empty 返回当前项是否为空
//
//	如果 Type 为 obj, 值为 {}, 返回 true
//	如果 Type 为 arr, 值为 [], 返回 true
//	如果 Type 为 val, 值为 nil 或 "", 返回 true
//	其余情况, 返回 false
func (i *Item) Empty() bool {
	switch i.jType {
	case JsonTypeArr:
		return len(i.arr) == 0
	case JsonTypeObj:
		return len(i.obj) == 0
	default:
		if i.val == nil {
			return true
		}
		if v, ok := i.val.(string); ok {
			return v == ""
		}
		return false
	}
}

// Ti 将当前对象转换为 TempItem 对象
func (i *Item) Ti() *TempItem {
	return &TempItem{item: i}
}
