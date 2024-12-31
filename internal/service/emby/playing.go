package emby

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/https"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/jsons"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/randoms"
	"github.com/gin-gonic/gin"
)

// PlayingStoppedHelper 拦截停止播放接口, 然后手动请求一次 Progress 接口记录进度
func PlayingStoppedHelper(c *gin.Context) {
	// 1 取出原始请求体信息
	bodyBytes, err := https.ExtractReqBody(c)
	if checkErr(c, err) {
		return
	}
	bodyJson, err := jsons.New(string(bodyBytes))
	if checkErr(c, err) {
		return
	}
	newSessionId := randoms.RandomHex(32)
	bodyJson.Put("PlaySessionId", jsons.NewByVal(newSessionId))
	c.Request.Body = io.NopCloser(bytes.NewBuffer([]byte(bodyJson.String())))

	// 2 提取 api apiKey
	kType, kName, apiKey := getApiKey(c)

	// 3 代理原始 Stopped 接口
	ProxyOrigin(c)

	// 4 代理 Progress 接口
	go func() {
		newBody := jsons.NewEmptyObj()
		newBody.Put("ItemId", jsons.NewByVal(bodyJson.Attr("ItemId").Val()))
		newBody.Put("PlaySessionId", jsons.NewByVal(newSessionId))
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
