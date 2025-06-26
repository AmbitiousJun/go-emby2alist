package emby

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"

	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/config"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/https"

	"github.com/gin-gonic/gin"
)

// ResortEpisodes 代理剧集列表请求
//
// 如果开启了 emby.episodes-unplay-prior 配置,
// 则会将未播剧集排在前面位置
func ResortEpisodes(c *gin.Context) {
	// 1 检查配置是否开启
	if !config.C.Emby.EpisodesUnplayPrior {
		checkErr(c, https.ProxyPass(c, config.C.Emby.Host))
		return
	}

	// 2 去除分页限制
	q := c.Request.URL.Query()
	q.Del("Limit")
	q.Del("StartIndex")
	c.Request.URL.RawQuery = q.Encode()

	// 3 代理请求
	c.Request.Header.Del("Accept-Encoding")
	resp, err := https.ProxyRequest(c, config.C.Emby.Host)
	if checkErr(c, err) {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		checkErr(c, errors.New(resp.Status))
		return
	}

	// 4 处理数据
	bodyBytes, err := io.ReadAll(resp.Body)
	if checkErr(c, err) {
		return
	}
	var ih ItemsHolder
	if err = json.Unmarshal(bodyBytes, &ih); checkErr(c, err) {
		return
	}
	resp.Header.Del("Content-Length")
	https.CloneHeader(c, resp.Header)
	defer func() {
		bytes, _ := json.Marshal(ih)
		c.Header("Content-Length", strconv.Itoa(len(bytes)))
		c.Writer.Write(bytes)
	}()

	if len(ih.Items) == 0 {
		return
	}

	type ValueInner struct {
		UserData struct {
			Played bool
		}
	}
	playedItems, allItems := make([]json.RawMessage, 0), make([]json.RawMessage, 0)
	for idx, value := range ih.Items {
		if len(allItems) > 0 {
			// 找到第一个未播的剧集之后, 剩余剧集都当作是未播的
			allItems = slices.Concat(allItems, ih.Items[idx:])
			break
		}

		var vi ValueInner
		if err := json.Unmarshal(value, &vi); err != nil {
			allItems = append(allItems, value)
			continue
		}

		if vi.UserData.Played {
			playedItems = append(playedItems, value)
			continue
		}
		allItems = append(allItems, value)
	}

	// 将已播的数据放在末尾
	allItems = append(allItems, playedItems...)
	ih.Items = allItems
}
