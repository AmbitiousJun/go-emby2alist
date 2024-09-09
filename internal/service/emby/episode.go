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
	// limitStr := c.Query("Limit")
	// limit, err := strconv.Atoi(limitStr)
	// if err != nil {
	// 	limit = math.MaxInt32
	// }
	// startStr := c.Query("StartIndex")
	// start, err := strconv.Atoi(startStr)
	// if err != nil {
	// 	start = 0
	// }
	q := c.Request.URL.Query()
	// 设置一个足够大的值, 查出全部数据
	// q.Set("Limit", strconv.Itoa(math.MaxInt32))
	// q.Set("StartIndex", "0")
	q.Del("Limit")
	q.Del("StartIndex")
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
	if !ok || items.Type() != jsons.JsonTypeArr {
		return
	}
	playedItems, allItems := make([]interface{}, 0), make([]interface{}, 0)
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

	// // 总个数小于等于 start, 返回 空
	// if len(allItems) <= start {
	// 	allItems = make([]interface{}, 0)
	// } else {
	// 	allItems = allItems[start:]
	// }

	// // 个数超过 Limit, 需要切断
	// if len(allItems) > limit {
	// 	allItems = allItems[:limit]
	// }

	resJson.Put("Items", jsons.NewByVal(allItems))
	c.Writer.Header().Del("Content-Length")
}
