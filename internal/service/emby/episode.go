package emby

import (
	"errors"
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/util/https"
	"go-emby2alist/internal/util/jsons"
	"net/http"

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
	c.Request.URL.RawQuery = q.Encode()

	// 3 代理请求
	res, respHeader := RawFetch(c.Request.URL.String(), c.Request.Method, c.Request.Body)
	if res.Code != http.StatusOK {
		checkErr(c, errors.New(res.Msg))
		return
	}
	resJson := res.Data
	https.CloneHeader(c, respHeader)
	defer func() {
		c.JSON(res.Code, resJson.Struct())
	}()

	// 4 处理数据
	items, ok := resJson.Attr("Items").Done()
	if !ok {
		return
	}
	playedItems, allItems := make([]*jsons.Item, 0), jsons.NewEmptyArr()
	items.RangeArr(func(_ int, value *jsons.Item) error {
		if allItems.Len() > 0 {
			// 找到第一个未播的剧集之后, 剩余剧集都当作是未播的
			allItems.Append(value)
			return nil
		}

		if played, ok := value.Attr("UserData").Attr("Played").Bool(); ok && played {
			playedItems = append(playedItems, value)
			return nil
		}

		allItems.Append(value)
		return nil
	})
	// 将已播的数据放在末尾
	allItems.Append(playedItems...)
	resJson.Put("Items", allItems)
}
