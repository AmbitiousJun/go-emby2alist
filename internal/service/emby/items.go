package emby

import (
	"encoding/json"
	"fmt"
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/util/colors"
	"go-emby2alist/internal/util/https"
	"go-emby2alist/internal/web/cache"
	"io"
	"log"
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

	// writeRespErr 响应客户端, 根据 err 自动判断
	// 如果 err 不为空, 直接使用原始 bodyBytes
	writeRespErr := func(err error, respBody []byte) {
		if err != nil {
			log.Printf(colors.ToRed("随机排序接口非预期响应, err: %v, 返回原始响应"), err)
			respBody = bodyBytes
		}
		c.Status(resp.StatusCode)
		resp.Header.Del("Content-Length")
		https.CloneHeader(c, resp.Header)
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

	// 准备一个相同大小的整型切片, 只存索引
	// 对索引重排序后再依据新的索引位置调整 item 的位置
	idxArr := make([]int, itemLen)
	for idx := range idxArr {
		idxArr[idx] = idx
	}
	rand.Shuffle(len(idxArr), func(i, j int) {
		idxArr[i], idxArr[j] = idxArr[j], idxArr[i]
	})
	newItems := make([]json.RawMessage, 0)
	for _, newIdx := range idxArr {
		newItems = append(newItems, resItems[newIdx])
	}

	newItemsBytes, _ := json.Marshal(newItems)
	resMain["Items"] = newItemsBytes
	newBodyBytes, _ := json.Marshal(resMain)
	writeRespErr(nil, newBodyBytes)
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
