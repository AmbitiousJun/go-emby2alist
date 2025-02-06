package emby

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/constant"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/https"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/jsons"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/randoms"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/strs"
	"github.com/gin-gonic/gin"
)

// stoppedHelperMarkMap 用于实现延时发送辅助 Progress 请求
var stoppedHelperMarkMap = sync.Map{}

// PlayingStoppedHelper 拦截停止播放接口, 然后手动请求一次 Progress 接口记录进度
func PlayingStoppedHelper(c *gin.Context) {
	// 取出原始请求体信息
	bodyBytes, err := https.ExtractReqBody(c)
	if checkErr(c, err) {
		return
	}
	bodyJson, err := jsons.New(string(bodyBytes))
	if checkErr(c, err) {
		return
	}

	// 代理原始 Stopped 接口
	ProxyOrigin(c)

	// 提取 api apiKey
	kType, kName, apiKey := getApiKey(c)

	// 至少播放 5 分钟才记录进度
	positionTicks, ok := bodyJson.Attr("PositionTicks").Int64()
	var minPos int64 = 5 * 60 * 10_000_000
	if !ok || positionTicks < minPos {
		return
	}

	go func() {
		// 依据 itemId 获取到内存中是否已经有待发送的辅助请求
		// 如果已存在, 就中止本次的辅助请求
		itemId, _ := bodyJson.Attr("ItemId").String()
		if itemIdNum, ok := bodyJson.Attr("ItemId").Int(); ok {
			itemId = strconv.Itoa(itemIdNum)
		}
		if strs.AnyEmpty(itemId) {
			return
		}

		// 每个请求使用一个随机数进行标记区分
		randomKey := randoms.RandomHex(32)
		stoppedHelperMarkMap.Store(itemId, randomKey)
		time.Sleep(30 * time.Second)
		if deleted := stoppedHelperMarkMap.CompareAndDelete(itemId, randomKey); !deleted {
			return
		}

		// 代理 Progress 接口
		newBody := jsons.NewEmptyObj()
		newBody.Put("ItemId", jsons.NewByVal(itemId))
		newBody.Put("PlaySessionId", jsons.NewByVal(randoms.RandomHex(32)))
		newBody.Put("PositionTicks", jsons.NewByVal(bodyJson.Attr("PositionTicks").Val()))
		log.Printf(colors.ToGray("开始发送辅助 Progress 进度记录, 内容: %v"), newBody)
		remote := config.C.Emby.Host + "/emby/Sessions/Playing/Progress"
		header := make(http.Header)
		header.Set("Content-Type", "application/json")
		if kType == Query {
			remote += fmt.Sprintf("?%s=%s", kName, apiKey)
		} else {
			header.Set(kName, apiKey)
		}
		resp, err := https.Request(http.MethodPost, remote, header, io.NopCloser(bytes.NewBuffer([]byte(newBody.String()))))
		if err != nil {
			log.Printf(colors.ToYellow("辅助发送 Progress 请求失败: %v"), err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			log.Printf(colors.ToYellow("辅助发送 Progress 请求失败, 源服务器返回状态码: %v"), resp.StatusCode)
			return
		}
		log.Println(colors.ToGreen("辅助发送 Progress 进度记录成功"))
	}()
}

// PlayingProgressHelper 拦截 Progress 请求, 如果进度报告为 0, 认为是无效请求
func PlayingProgressHelper(c *gin.Context) {
	// 取出原始请求体信息
	bodyBytes, err := https.ExtractReqBody(c)
	if checkErr(c, err) {
		return
	}
	bodyJson, err := jsons.New(string(bodyBytes))
	if checkErr(c, err) {
		return
	}

	if pt, ok := bodyJson.Attr("PositionTicks").Int64(); ok && pt <= 10_000_000 {
		c.Status(http.StatusNoContent)
		return
	}
	ProxyOrigin(c)
}

// PlayedItemsIntercepter 拦截剧集标记请求, 中断辅助请求
func PlayedItemsIntercepter(c *gin.Context) {
	// 解析 itemId
	routeMatches := c.GetStringSlice(constant.RouteSubMatchGinKey)
	if len(routeMatches) < 2 {
		return
	}
	itemId := routeMatches[1]

	// 移除内存中的辅助请求标记
	stoppedHelperMarkMap.Delete(itemId)

	ProxyOrigin(c)
}
