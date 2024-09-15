package emby

import (
	"errors"
	"fmt"
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/util/https"
	"go-emby2alist/internal/util/jsons"
	"go-emby2alist/internal/web/cache"
	"io"
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
	if !ok {
		checkErr(c, errors.New("非预期响应"))
		return
	}
	items.Shuffle()
	c.JSON(http.StatusOK, resJson.Struct())
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
		c.Header(cache.HeaderKeyExpired, cache.Duration(time.Hour*12))
		c.JSON(res.Code, res.Data.Struct())
	}
}
