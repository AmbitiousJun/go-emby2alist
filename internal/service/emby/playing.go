package emby

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/constant"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/https"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/jsons"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/randoms"
	"github.com/gin-gonic/gin"
)

// stoppedHelperSentTimerMap 用于实现延时发送辅助 Progress 请求
var stoppedHelperSentTimerMap = sync.Map{}

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
		// 依据 itemId 获取到内存中是否已经有待发送的辅助请求计时器
		// 如果已存在, 就停止计时器, 以当前请求的进度为主
		itemId, ok := bodyJson.Attr("ItemId").String()
		if !ok {
			return
		}
		oldTimer, oldTimerExist := stoppedHelperSentTimerMap.Load(itemId)
		if oldTimerExist {
			if timer, ok := oldTimer.(*time.Timer); ok && timer != nil {
				timer.Stop()
			}
		}

		newTimer := time.NewTimer(30 * time.Second)
		timeoutTimer := time.NewTimer(40 * time.Second)
		if !oldTimerExist {
			// 不存在旧定时器, 尝试设置新定时器
			if _, loaded := stoppedHelperSentTimerMap.LoadOrStore(itemId, newTimer); loaded {
				// 已经有其他请求正在处理该 itemId
				return
			}
		} else {
			// 存在旧定时器, 尝试替换为新定时器
			if ok := stoppedHelperSentTimerMap.CompareAndSwap(itemId, oldTimer, newTimer); !ok {
				// 如果 CAS 失败, 说明已经有其他请求在处理该 itemId, 则取消发送辅助请求
				return
			}
		}

		// 等待计时器触发后再执行逻辑
		select {
		case <-timeoutTimer.C:
			return
		case <-newTimer.C:
			stoppedHelperSentTimerMap.CompareAndDelete(itemId, newTimer)
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

	// 取出内存中是否有待发送的辅助请求计时器
	if timer, ok := stoppedHelperSentTimerMap.LoadAndDelete(itemId); ok {
		if t, ok := timer.(*time.Timer); ok && t != nil {
			t.Stop()
		}
	}

	ProxyOrigin(c)
}
