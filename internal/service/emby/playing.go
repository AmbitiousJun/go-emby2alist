package emby

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/config"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/https"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/jsons"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/randoms"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/strs"
	"github.com/gin-gonic/gin"
)

// PlayingStoppedHelper 拦截停止播放接口, 然后手动请求一次 Progress 接口记录进度
func PlayingStoppedHelper(c *gin.Context) {
	// 取出原始请求体信息
	bodyBytes, newBody, err := https.ExtractReqBody(c.Request.Body)
	if checkErr(c, err) {
		return
	}
	c.Request.Body = newBody
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

	// 发送辅助请求记录播放进度
	itemId, _ := bodyJson.Attr("ItemId").String()
	if itemIdNum, ok := bodyJson.Attr("ItemId").Int(); ok {
		itemId = strconv.Itoa(itemIdNum)
	}
	if strs.AnyEmpty(itemId) {
		return
	}
	body := jsons.NewEmptyObj()
	body.Put("ItemId", jsons.FromValue(itemId))
	body.Put("PlaySessionId", jsons.FromValue(randoms.RandomHex(32)))
	body.Put("PositionTicks", jsons.FromValue(bodyJson.Attr("PositionTicks").Val()))
	go sendPlayingProgress(kType, kName, apiKey, body)
}

// PlayingProgressHelper 拦截 Progress 请求, 如果进度报告为 0, 认为是无效请求
func PlayingProgressHelper(c *gin.Context) {
	// 取出原始请求体信息
	bodyBytes, newBody, err := https.ExtractReqBody(c.Request.Body)
	if checkErr(c, err) {
		return
	}
	c.Request.Body = newBody
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

// sendPlayingProgress 发送辅助播放进度请求
func sendPlayingProgress(kType ApiKeyType, kName, apiKey string, body *jsons.Item) {
	if body == nil {
		return
	}

	inner := func(remote string) error {
		header := make(http.Header)
		header.Set("Content-Type", "application/json")
		if kType == Query {
			remote += fmt.Sprintf("?%s=%s", kName, apiKey)
		} else {
			header.Set(kName, apiKey)
		}
		resp, err := https.Post(remote).Header(header).Body(io.NopCloser(bytes.NewBuffer([]byte(body.String())))).Do()
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("源服务器返回错误状态: %v", resp.Status)
		}
		return nil
	}

	log.Printf(colors.ToGray("开始发送辅助 Progress 进度记录, 内容: %v"), body)
	if err := inner(config.C.Emby.Host + "/emby/Sessions/Playing/Progress"); err != nil {
		log.Printf(colors.ToYellow("辅助发送 Progress 进度记录失败: %v"), err)
		return
	}
	if err := inner(config.C.Emby.Host + "/emby/Sessions/Playing/Stopped"); err != nil {
		log.Printf(colors.ToYellow("辅助发送 Progress 进度记录失败: %v"), err)
		return
	}
	log.Println(colors.ToGreen("辅助发送 Progress 进度记录成功"))
}
