package emby

import (
	"errors"
	"fmt"
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/util/https"
	"go-emby2alist/internal/util/jsons"
	"go-emby2alist/internal/web/cache"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ResortRandomItems 对随机的 items 列表进行重排序
func ResortRandomItems(c *gin.Context) {
	// 1 如果没有开启配置, 代理原请求并返回
	if !config.C.Emby.ResortRandomItems {
		if res, ok := proxyAndSetRespHeader(c); ok {
			c.JSON(res.Code, res.Data.Struct())
		}
		return
	}

	// 2 请求去除个数限制后的原始列表
	u := strings.ReplaceAll(https.ClientRequestUrl(c), "/Items", "/Items/no_limit")
	resp, err := https.Request(http.MethodGet, u, c.Request.Header, c.Request.Body)
	if checkErr(c, err) {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		checkErr(c, fmt.Errorf("错误的响应码: %d", resp.StatusCode))
		return
	}

	// 3 转换 json 响应
	bodyBytes, err := io.ReadAll(resp.Body)
	if checkErr(c, err) {
		return
	}
	resJson, err := jsons.New(string(bodyBytes))
	if checkErr(c, err) {
		return
	}

	// 4 取出 Items 进行重排序
	items, ok := resJson.Attr("Items").Done()
	if !ok || items.Type() != jsons.JsonTypeArr {
		checkErr(c, errors.New("非预期响应"))
		return
	}
	defer func() {
		c.JSON(http.StatusOK, resJson.Struct())
	}()
	if items.Empty() {
		return
	}

	// 准备一个相同大小的整型切片, 只存索引
	// 对索引重排序后再依据新的索引位置调整 item 的位置
	idxArr := make([]int, items.Len())
	for idx := range idxArr {
		idxArr[idx] = idx
	}
	rand.Shuffle(len(idxArr), func(i, j int) {
		idxArr[i], idxArr[j] = idxArr[j], idxArr[i]
	})
	newItems := jsons.NewEmptyArr()
	for _, newIdx := range idxArr {
		newItems.Append(items.ValuesArr()[newIdx])
	}
	resJson.Put("Items", newItems)
}

// RandomItemsWithLimit 代理原始的随机列表接口
// 个数限制为 700
func RandomItemsWithLimit(c *gin.Context) {
	u := c.Request.URL
	u.Path = strings.TrimSuffix(u.Path, "/no_limit")
	q := u.Query()
	q.Set("Limit", "700")
	u.RawQuery = q.Encode()
	res, ok := proxyAndSetRespHeader(c)
	if ok {
		c.Writer.Header().Del("Content-Length")
		c.Header(cache.HeaderKeyExpired, cache.Duration(time.Hour))
		c.JSON(res.Code, res.Data.Struct())
	}
}
