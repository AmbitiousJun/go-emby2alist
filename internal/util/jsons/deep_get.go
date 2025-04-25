package jsons

// TempItem 临时暂存 Item 对象
type TempItem struct {

	// item 临时对象
	item *Item
}

// Attr 获取对象 item 某个 key 的值
// 如果需要立即获得 *Item 对象, 需要链式调用 Done() 方法获取
func (ti *TempItem) Attr(key string) *TempItem {
	if ti.item == nil {
		return ti
	}
	if ti.item.jType != JsonTypeObj {
		ti.item = nil
		return ti
	}
	if subI, ok := ti.item.obj[key]; ok {
		ti.item = subI
	} else {
		ti.item = nil
	}
	return ti
}

// Idx 获取数组 item 某个 index 的值
// 如果需要立即获得 *Item 对象, 需要链式调用 Done() 方法获取
func (ti *TempItem) Idx(index int) *TempItem {
	if ti.item == nil {
		return ti
	}
	if ti.item.jType != JsonTypeArr {
		ti.item = nil
		return ti
	}
	if index < 0 || index >= len(ti.item.arr) {
		ti.item = nil
		return ti
	}
	ti.item = ti.item.arr[index]
	return ti
}

// Done 获取链式调用后的 JSON 值
func (ti *TempItem) Done() (*Item, bool) {
	if ti.item == nil {
		return nil, false
	}
	return ti.item, true
}

// Bool 获取链式调用后的 bool 值
func (ti *TempItem) Bool() (bool, bool) {
	if ti.item == nil || ti.item.jType != JsonTypeVal {
		return false, false
	}
	if val, ok := ti.item.val.(bool); ok {
		return val, true
	}
	return false, false
}

// Int 获取链式调用后的 int 值
func (ti *TempItem) Int() (int, bool) {
	if ti.item == nil || ti.item.jType != JsonTypeVal {
		return 0, false
	}
	if val, ok := ti.item.val.(int); ok {
		return val, true
	}
	return 0, false
}

// Int64 获取链式调用后的 int64 值
func (ti *TempItem) Int64() (int64, bool) {
	if ti.item == nil || ti.item.jType != JsonTypeVal {
		return 0, false
	}
	if val, ok := ti.item.val.(int); ok {
		return int64(val), true
	}
	if val, ok := ti.item.val.(int64); ok {
		return val, true
	}
	return 0, false
}

// Float 获取链式调用后的 float 值
func (ti *TempItem) Float() (float64, bool) {
	if ti.item == nil || ti.item.jType != JsonTypeVal {
		return 0, false
	}
	if val, ok := ti.item.val.(float64); ok {
		return val, true
	}
	return 0, false
}

// String 获取链式调用后的 string 值
func (ti *TempItem) String() (string, bool) {
	if ti.item == nil || ti.item.jType != JsonTypeVal {
		return "", false
	}
	if val, ok := ti.item.val.(string); ok {
		return val, true
	}
	return "", false
}

// Val 获取链式调用后的 val 值, 类型不匹配时返回 nil
func (ti *TempItem) Val() any {
	if ti.item == nil || ti.item.jType != JsonTypeVal {
		return nil
	}
	return ti.item.val
}

// Set 设置当前链式调用后的 val 值, 类型不匹配时不作更改
func (ti *TempItem) Set(val any) *TempItem {
	if ti.item == nil || ti.item.jType != JsonTypeVal {
		return ti
	}

	if val == nil {
		ti.item.val = nil
		return ti
	}

	switch val.(type) {
	case bool, string, int, float64, int64:
		ti.item.val = val
	default:
	}

	return ti
}
