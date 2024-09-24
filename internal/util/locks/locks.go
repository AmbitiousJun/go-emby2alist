package locks

import (
	"sync"

	"github.com/AmbitiousJun/go-emby2alist/internal/util/strs"
)

// mutexMap 统一存放锁
var mutexMap = sync.Map{}

// Mutex 根据指定的 key 获取锁
func Mutex(key string) *sync.Mutex {
	if strs.AnyEmpty(key) {
		return nil
	}
	mu, _ := mutexMap.LoadOrStore(key, new(sync.Mutex))
	return mu.(*sync.Mutex)
}

// Del 删除指定 key 的锁对象
func Del(key string) {
	if strs.AnyEmpty(key) {
		return
	}
	mutexMap.Delete(key)
}
