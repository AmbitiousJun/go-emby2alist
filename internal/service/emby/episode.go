package emby

import (
	"errors"
	"net/http"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/https"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/jsons"

	"github.com/gin-gonic/gin"
)

// ResortEpisodes 代理剧集列表请求
//
// 如果开启了 emby.episodes-unplay-prior 配置,
// 则会将未播剧集排在前面位置
func ResortEpisodes(c *gin.Context) {
	// 1 检查配置是否开启
	if !config.C.Emby.EpisodesUnplayPrior {
		checkErr(c, https.ProxyRequest(c, config.C.Emby.Host, true))
		return
	}

	// 2 去除分页限制
	q := c.Request.URL.Query()
	q.Del("Limit")
	q.Del("StartIndex")
	c.Request.URL.RawQuery = q.Encode()

	// 3 代理请求
	c.Request.Header.Del("Accept-Encoding")
	res, respHeader := RawFetch(c.Request.URL.String(), c.Request.Method, c.Request.Header, c.Request.Body)
	if res.Code != http.StatusOK {
		checkErr(c, errors.New(res.Msg))
		return
	}
	resJson := res.Data
	https.CloneHeader(c, respHeader)
	defer func() {
		jsons.OkResp(c, resJson)
	}()

	// 4 处理数据
	items, ok := resJson.Attr("Items").Done()
	if !ok || items.Type() != jsons.JsonTypeArr {
		return
	}
	playedItems, allItems := make([]*jsons.Item, 0), make([]*jsons.Item, 0)
	items.RangeArr(func(_ int, value *jsons.Item) error {
		if len(allItems) > 0 {
			// 找到第一个未播的剧集之后, 剩余剧集都当作是未播的
			allItems = append(allItems, value)
			return nil
		}

		if played, ok := value.Attr("UserData").Attr("Played").Bool(); ok && played {
			playedItems = append(playedItems, value)
			return nil
		}

		allItems = append(allItems, value)
		return nil
	})

	// 将已播的数据放在末尾
	allItems = append(allItems, playedItems...)

	resJson.Put("Items", jsons.NewByVal(allItems))
	c.Writer.Header().Del("Content-Length")
}
