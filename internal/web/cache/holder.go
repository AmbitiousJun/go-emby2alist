package cache

import (
	"bytes"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/config"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/strs"

	"github.com/gin-gonic/gin"
)

const (

	// MaxCacheSize 缓存最大大小 (Byte)
	//
	// 这里的大小指的是响应体大小, 实际占用大小可能略大一些
	MaxCacheSize int64 = 100 * 1024 * 1024

	// MaxCacheNum 最多缓存多少个请求信息
	MaxCacheNum = 8092

	// HeaderKeyExpired 缓存过期响应头, 用于覆盖默认的缓存过期时间
	HeaderKeyExpired = "Expired"
)

// currentCacheSize 当前内存中的缓存大小 (Byte)
var currentCacheSize int64 = 0

// DefaultExpired 默认的请求过期时间
//
// 可通过设置 "Expired" 响应头进行覆盖
var DefaultExpired = func() time.Duration { return config.C.Cache.ExpiredDuration() }

// cacheMap 存放缓存数据的 map
var cacheMap = sync.Map{}

// preCacheChan 预缓存通道
//
// 缓存数据先暂存在通道中, 再由专门的 goroutine 单线程处理
//
// preCacheChan 的淘汰规则是先入先淘汰, 不管缓存对象的过期时间
var preCacheChan = make(chan *respCache, MaxCacheNum)

// cacheHandleWaitGroup 允许等待预缓存通道处理完毕后再获取数据
var cacheHandleWaitGroup = sync.WaitGroup{}

func init() {
	go loopMaintainCache()
}

// loopMaintainCache cacheMap 由单独的 goroutine 维护
func loopMaintainCache() {

	// cleanCache 清洗缓存数据
	cleanCache := func() {
		validCnt := 0
		nowMillis := time.Now().UnixMilli()
		toDelete := make([]*respCache, 0)

		cacheMap.Range(func(key, value any) bool {
			rc := value.(*respCache)
			if nowMillis > rc.expired || validCnt == MaxCacheNum || currentCacheSize > MaxCacheSize {
				toDelete = append(toDelete, rc)
			} else {
				validCnt++
			}
			return true
		})

		for _, rc := range toDelete {
			cacheMap.Delete(rc.cacheKey)
			currentCacheSize -= int64(len(rc.body))
			delSpaceCache(rc.header.space, rc.header.spaceKey)
		}
	}

	// putrespCache 将缓存对象维护到 cacheMap 中
	//
	// 同时淘汰掉过期缓存
	putrespCache := func(rc *respCache) {
		cacheMap.Store(rc.cacheKey, rc)
		currentCacheSize += int64(len(rc.body))
		space, spaceKey := rc.header.space, rc.header.spaceKey
		if strs.AllNotEmpty(space, spaceKey) {
			putSpaceCache(space, spaceKey, rc)
			log.Printf(colors.ToGreen("刷新缓存空间, space: %s, spaceKey: %s"), space, spaceKey)
		}
	}

	timer := time.NewTicker(time.Second * 10)
	defer timer.Stop()
	for {
		select {
		case rc := <-preCacheChan:
			putrespCache(rc)
			cacheHandleWaitGroup.Done()
		case <-timer.C:
			cleanCache()
		}
	}
}

// getCache 根据 cacheKey 获取缓存
func getCache(cacheKey string) (*respCache, bool) {
	if c, ok := cacheMap.Load(cacheKey); ok {
		return c.(*respCache), true
	}
	return nil, false
}

// putCache 设置缓存
func putCache(cacheKey string, c *gin.Context, respBody *bytes.Buffer, respHeader respHeader) {
	if cacheKey == "" || c == nil || respBody == nil {
		return
	}

	// 计算缓存过期时间
	nowMillis := time.Now().UnixMilli()
	expiredMillis := int64(DefaultExpired()) + nowMillis
	if expiredNum, err := strconv.Atoi(respHeader.expired); err == nil {
		customMillis := int64(expiredNum)

		// 特定接口不使用缓存
		if customMillis < 0 {
			return
		}

		if customMillis > nowMillis {
			expiredMillis = customMillis
		}
	}

	rc := &respCache{
		code:     c.Writer.Status(),
		body:     respBody.Bytes(),
		cacheKey: cacheKey,
		expired:  expiredMillis,
		header:   respHeader,
	}

	// 依据先进先淘汰原则, 将最新缓存放入预缓存通道中
	cacheHandleWaitGroup.Add(1)
	doneOnce := sync.OnceFunc(cacheHandleWaitGroup.Done)
	for {
		select {
		case preCacheChan <- rc:
			return
		default:
			log.Println("预缓存通道已满, 淘汰旧缓存")
			<-preCacheChan
			doneOnce()
		}
	}
}
