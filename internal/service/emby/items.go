package emby

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/config"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/https"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/jsons"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/urls"
	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/web/cache"

	"github.com/gin-gonic/gin"
)

const (

	// ItemsCacheSpace 专门存放 items 信息的缓存空间
	ItemsCacheSpace = "UserItems"

	// ResortMinNum 至少请求多少个 item 时才会走重排序逻辑
	ResortMinNum = 300
)

// ResortRandomItems 对随机的 items 列表进行重排序
func ResortRandomItems(c *gin.Context) {
	// 如果没有开启配置, 代理原请求并返回
	if !config.C.Emby.ResortRandomItems {
		ProxyOrigin(c)
		return
	}

	// 如果请求的个数较少, 认为不是随机播放列表, 代理原请求并返回
	limit, err := strconv.Atoi(c.Query("Limit"))
	if err == nil && limit < ResortMinNum {
		ProxyOrigin(c)
		return
	}

	// 从缓存空间中获取列表
	spaceCache, ok := cache.GetSpaceCache(ItemsCacheSpace, calcRandomItemsCacheKey(c))

	// 缓存空间没有数据时, 默认使用 emby 的原始随机结果
	if !ok {
		u := strings.ReplaceAll(https.ClientRequestUrl(c.Request), "/Items", "/Items/with_limit")
		c.Redirect(http.StatusTemporaryRedirect, u)
		return
	}

	bodyBytes := spaceCache.BodyBytes()
	code := spaceCache.Code()
	header := spaceCache.Headers()
	log.Println(colors.ToBlue("使用缓存空间中的 random items 列表"))

	// 响应客户端, 根据 err 自动判断
	// 如果 err 不为空, 使用原始 bodyBytes
	err = nil
	var ih ItemsHolder
	defer func() {
		respBody, _ := json.Marshal(ih)
		if err != nil {
			log.Printf(colors.ToRed("随机排序接口非预期响应, err: %v, 返回原始响应"), err)
			respBody = bodyBytes
		}

		c.Status(code)
		header.Set("Content-Length", strconv.Itoa(len(respBody)))
		https.CloneHeader(c.Writer, header)
		c.Writer.Write(respBody)
	}()

	// 对 item 内部结构不关心, 故使用原始的 json 序列化提高处理速度
	if err = json.Unmarshal(bodyBytes, &ih); err != nil {
		return
	}

	itemLen := len(ih.Items)
	if itemLen == 0 {
		return
	}
	rand.Shuffle(itemLen, func(i, j int) {
		ih.Items[i], ih.Items[j] = ih.Items[j], ih.Items[i]
	})
}

// RandomItemsWithLimit 代理原始的随机列表接口
func RandomItemsWithLimit(c *gin.Context) {
	u := c.Request.URL
	u.Path = strings.TrimSuffix(u.Path, "/with_limit")
	q := u.Query()
	q.Set("Limit", "500")
	q.Del("SortOrder")
	u.RawQuery = q.Encode()
	embyHost := config.C.Emby.Host
	c.Request.Header.Del("Accept-Encoding")
	resp, err := https.Request(c.Request.Method, embyHost+u.String()).
		Header(c.Request.Header).
		Body(c.Request.Body).
		Do()
	if checkErr(c, err) {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		checkErr(c, fmt.Errorf("错误的响应码: %v", resp.StatusCode))
		return
	}

	c.Status(resp.StatusCode)
	https.CloneHeader(c.Writer, resp.Header)
	c.Header(cache.HeaderKeyExpired, cache.Duration(time.Hour*3))
	c.Header(cache.HeaderKeySpace, ItemsCacheSpace)
	c.Header(cache.HeaderKeySpaceKey, calcRandomItemsCacheKey(c))
	io.Copy(c.Writer, resp.Body)
}

// calcRandomItemsCacheKey 计算 random items 在缓存空间中的 key 值
func calcRandomItemsCacheKey(c *gin.Context) string {
	return c.Query("IncludeItemTypes") +
		c.Query("Recursive") +
		c.Query("Fields") +
		c.Query("EnableImageTypes") +
		c.Query("ImageTypeLimit") +
		c.Query("IsFavorite") +
		c.Query("IsFolder") +
		c.Query("ProjectToMedia") +
		c.Query("ParentId")
}

// ProxyAddItemsPreviewInfo 代理 Items 接口, 并附带上转码版本信息
func ProxyAddItemsPreviewInfo(c *gin.Context) {
	// 代理请求
	c.Request.Header.Del("Accept-Encoding")
	resp, err := https.ProxyRequest(c.Request, config.C.Emby.Host)
	if checkErr(c, err) {
		return
	}
	defer resp.Body.Close()

	// 检查响应, 读取为 JSON
	if resp.StatusCode != http.StatusOK {
		checkErr(c, fmt.Errorf("emby 远程返回了错误的响应码: %d", resp.StatusCode))
		return
	}
	resJson, err := jsons.Read(resp.Body)
	if checkErr(c, err) {
		return
	}

	// 预响应请求
	defer func() {
		https.CloneHeader(c.Writer, resp.Header)
		jsons.OkResp(c.Writer, resJson)
	}()

	// 获取 Items 数组
	itemsArr, ok := resJson.Attr("Items").Done()
	if !ok || itemsArr.Empty() || itemsArr.Type() != jsons.JsonTypeArr {
		return
	}

	// 遍历每个 Item, 修改 MediaSource 信息
	proresMediaStreams, _ := jsons.New(`[{"AspectRatio":"16:9","AttachmentSize":0,"AverageFrameRate":25,"BitDepth":8,"BitRate":4838626,"Codec":"prores","CodecTag":"hev1","DisplayTitle":"4K HEVC","ExtendedVideoSubType":"None","ExtendedVideoSubTypeDescription":"None","ExtendedVideoType":"None","Height":2160,"Index":0,"IsDefault":true,"IsExternal":false,"IsForced":false,"IsHearingImpaired":false,"IsInterlaced":false,"IsTextSubtitleStream":false,"Language":"und","Level":150,"PixelFormat":"yuv420p","Profile":"Main","Protocol":"File","RealFrameRate":25,"RefFrames":1,"SupportsExternalStream":false,"TimeBase":"1/90000","Type":"Video","VideoRange":"SDR","Width":3840},{"AttachmentSize":0,"BitRate":124573,"ChannelLayout":"stereo","Channels":2,"Codec":"aac","CodecTag":"mp4a","DisplayTitle":"AAC stereo (默认)","ExtendedVideoSubType":"None","ExtendedVideoSubTypeDescription":"None","ExtendedVideoType":"None","Index":1,"IsDefault":true,"IsExternal":false,"IsForced":false,"IsHearingImpaired":false,"IsInterlaced":false,"IsTextSubtitleStream":false,"Language":"und","Profile":"LC","Protocol":"File","SampleRate":44100,"SupportsExternalStream":false,"TimeBase":"1/44100","Type":"Audio"}]`)
	itemsArr.RangeArr(func(index int, item *jsons.Item) error {
		mediaSources, ok := item.Attr("MediaSources").Done()
		if !ok || mediaSources.Empty() {
			return nil
		}

		toAdd := make([]*jsons.Item, 0)
		mediaSources.RangeArr(func(_ int, ms *jsons.Item) error {
			originId, _ := ms.Attr("Id").String()
			originName := findMediaSourceName(ms)
			allTplIds := getAllPreviewTemplateIds()
			ms.Put("Name", jsons.FromValue("(原画) "+originName))

			if path, ok := ms.Attr("Path").String(); ok {
				ms.Attr("Path").Set(urls.Unescape(path))
			}

			// 检查用户是否启用了转码版本获取
			if !config.C.VideoPreview.Enable {
				return nil
			}

			for _, tplId := range allTplIds {
				copyMs := jsons.FromValue(ms.Struct())
				copyMs.Put("Name", jsons.FromValue(fmt.Sprintf("(%s) %s", tplId, originName)))
				copyMs.Put("Id", jsons.FromValue(fmt.Sprintf("%s%s%s", originId, MediaSourceIdSegment, tplId)))
				copyMs.Put("MediaStreams", proresMediaStreams)
				toAdd = append(toAdd, copyMs)
			}
			return nil
		})

		mediaSources.Append(toAdd...)
		return nil
	})
}

// ProxyLatestItems 代理 Latest 请求
func ProxyLatestItems(c *gin.Context) {
	// 代理请求
	c.Request.Header.Del("Accept-Encoding")
	resp, err := https.ProxyRequest(c.Request, config.C.Emby.Host)
	if checkErr(c, err) {
		return
	}
	defer resp.Body.Close()

	// 检查响应, 读取为 JSON
	if resp.StatusCode != http.StatusOK {
		checkErr(c, fmt.Errorf("emby 远程返回了错误的响应码: %d", resp.StatusCode))
		return
	}
	resJson, err := jsons.Read(resp.Body)
	if checkErr(c, err) {
		return
	}

	// 预响应请求
	defer func() {
		https.CloneHeader(c.Writer, resp.Header)
		jsons.OkResp(c.Writer, resJson)
	}()

	// 遍历 MediaSources 解码 path
	if resJson.Type() != jsons.JsonTypeArr {
		return
	}
	resJson.RangeArr(func(_ int, item *jsons.Item) error {
		mediaSources, ok := item.Attr("MediaSources").Done()
		if !ok || mediaSources.Type() != jsons.JsonTypeArr || mediaSources.Empty() {
			return nil
		}
		mediaSources.RangeArr(func(_ int, ms *jsons.Item) error {
			if path, ok := ms.Attr("Path").String(); ok {
				ms.Attr("Path").Set(urls.Unescape(path))
			}
			return nil
		})
		return nil
	})

}
