package emby

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/https"
	"github.com/AmbitiousJun/go-emby2alist/internal/web/cache"

	"github.com/gin-gonic/gin"
)

const (

	// ItemsCacheSpace 专门存放 items 信息的缓存空间
	ItemsCacheSpace = "UserItems"

	// ResortMinNum 至少请求多少个 item 时才会走重排序逻辑
	ResortMinNum = 300
)

// ResortRandomItems 对随机的 items 列表进行重排序
func ResortRandomItems(c *gin.Context) {
	// 如果没有开启配置, 代理原请求并返回
	if !config.C.Emby.ResortRandomItems {
		ProxyOrigin(c)
		return
	}

	// 如果请求的个数较少, 认为不是随机播放列表, 代理原请求并返回
	limit, err := strconv.Atoi(c.Query("Limit"))
	if err == nil && limit < ResortMinNum {
		ProxyOrigin(c)
		return
	}

	// 优先从缓存空间中获取列表
	var code int
	var header http.Header
	var bodyBytes []byte
	spaceCache, ok := cache.GetSpaceCache(ItemsCacheSpace, calcRandomItemsCacheKey(c))
	if ok {
		bodyBytes = spaceCache.BodyBytes()
		code = spaceCache.Code()
		header = spaceCache.Headers()
		log.Println(colors.ToBlue("使用缓存空间中的 random items 列表"))
	} else {
		// 请求原始列表
		u := strings.ReplaceAll(https.ClientRequestUrl(c), "/Items", "/Items/with_limit")
		resp, err := https.Request(http.MethodGet, u, c.Request.Header, c.Request.Body)
		if checkErr(c, err) {
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			checkErr(c, fmt.Errorf("错误的响应码: %d", resp.StatusCode))
			return
		}

		// 转换 json 响应
		bodyBytes, err = io.ReadAll(resp.Body)
		if checkErr(c, err) {
			return
		}
		code = resp.StatusCode
		header = resp.Header
	}

	// writeRespErr 响应客户端, 根据 err 自动判断
	// 如果 err 不为空, 直接使用原始 bodyBytes
	writeRespErr := func(err error, respBody []byte) {
		if err != nil {
			log.Printf(colors.ToRed("随机排序接口非预期响应, err: %v, 返回原始响应"), err)
			respBody = bodyBytes
		}
		c.Status(code)
		header.Del("Content-Length")
		https.CloneHeader(c, header)
		c.Writer.Write(respBody)
	}

	// 对 item 内部结构不关心, 故使用原始的 json 序列化提高处理速度
	var resMain map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &resMain); err != nil {
		writeRespErr(err, nil)
		return
	}
	var resItems []json.RawMessage
	if err := json.Unmarshal(resMain["Items"], &resItems); err != nil {
		writeRespErr(err, nil)
		return
	}
	itemLen := len(resItems)
	if itemLen == 0 {
		writeRespErr(nil, bodyBytes)
		return
	}

	// idxChan 使用异步的方式生成随机索引塞到通道中
	idxChan := make(chan int, itemLen)
	go func() {
		idxArr := make([]int, itemLen)
		for idx := range idxArr {
			idxArr[idx] = idx
		}
		tot := itemLen
		for tot > 0 {
			randomIdx := rand.Intn(tot)
			idxChan <- idxArr[randomIdx]
			// 将当前元素与最后一个元素交换, 总个数 -1
			idxArr[tot-1], idxArr[randomIdx] = idxArr[randomIdx], idxArr[tot-1]
			tot--
		}
		close(idxChan)
	}()

	newItems := make([]json.RawMessage, 0)
	for newIdx := range idxChan {
		newItems = append(newItems, resItems[newIdx])
	}

	newItemsBytes, _ := json.Marshal(newItems)
	resMain["Items"] = newItemsBytes
	newBodyBytes, _ := json.Marshal(resMain)
	writeRespErr(nil, newBodyBytes)
}

// RandomItemsWithLimit 代理原始的随机列表接口
func RandomItemsWithLimit(c *gin.Context) {
	u := c.Request.URL
	u.Path = strings.TrimSuffix(u.Path, "/with_limit")
	q := u.Query()
	q.Set("Limit", "500")
	q.Del("SortOrder")
	u.RawQuery = q.Encode()
	embyHost := config.C.Emby.Host
	c.Request.Header.Del("Accept-Encoding")
	resp, err := https.Request(c.Request.Method, embyHost+u.String(), c.Request.Header, c.Request.Body)
	if checkErr(c, err) {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		checkErr(c, fmt.Errorf("错误的响应码: %v", resp.StatusCode))
		return
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if checkErr(c, err) {
		return
	}

	https.CloneHeader(c, resp.Header)
	c.Header(cache.HeaderKeyExpired, cache.Duration(time.Minute*15))
	c.Header(cache.HeaderKeySpace, ItemsCacheSpace)
	c.Header(cache.HeaderKeySpaceKey, calcRandomItemsCacheKey(c))
	c.Status(resp.StatusCode)
	c.Writer.Write(bodyBytes)
}

// calcRandomItemsCacheKey 计算 random items 在缓存空间中的 key 值
func calcRandomItemsCacheKey(c *gin.Context) string {
	return c.Query("IncludeItemTypes") +
		c.Query("Recursive") +
		c.Query("Fields") +
		c.Query("EnableImageTypes") +
		c.Query("ImageTypeLimit") +
		c.Query("IsFavorite") +
		c.Query("IsFolder") +
		c.Query("ProjectToMedia")
}
