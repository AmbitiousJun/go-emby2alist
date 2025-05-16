package emby

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/https"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/jsons"
	"github.com/AmbitiousJun/go-emby2alist/internal/web/cache"

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

	// 优先从缓存空间中获取列表
	var code int
	var header http.Header
	var bodyBytes []byte
	spaceCache, ok := cache.GetSpaceCache(ItemsCacheSpace, calcRandomItemsCacheKey(c))
	if ok {
		bodyBytes = spaceCache.BodyBytes()
		code = spaceCache.Code()
		header = spaceCache.Headers()
		log.Println(colors.ToBlue("使用缓存空间中的 random items 列表"))
	} else {
		// 请求原始列表
		u := strings.ReplaceAll(https.ClientRequestUrl(c), "/Items", "/Items/with_limit")
		resp, err := https.Request(http.MethodGet, u, c.Request.Header, c.Request.Body)
		if checkErr(c, err) {
			return
		}

		if resp.StatusCode != http.StatusOK {
			checkErr(c, fmt.Errorf("错误的响应码: %d", resp.StatusCode))
			return
		}

		// 转换 json 响应
		bodyBytes, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if checkErr(c, err) {
			return
		}
		code = resp.StatusCode
		header = resp.Header
	}

	// writeRespErr 响应客户端, 根据 err 自动判断
	// 如果 err 不为空, 直接使用原始 bodyBytes
	writeRespErr := func(err error, respBody []byte) {
		if err != nil {
			log.Printf(colors.ToRed("随机排序接口非预期响应, err: %v, 返回原始响应"), err)
			respBody = bodyBytes
		}
		c.Status(code)
		header.Del("Content-Length")
		https.CloneHeader(c, header)
		io.Copy(c.Writer, bytes.NewBuffer(respBody))
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

	rand.Shuffle(itemLen, func(i, j int) {
		resItems[i], resItems[j] = resItems[j], resItems[i]
	})

	newItemsBytes, _ := json.Marshal(resItems)
	resMain["Items"] = newItemsBytes
	newBodyBytes, _ := json.Marshal(resMain)
	writeRespErr(nil, newBodyBytes)
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
	resp, err := https.Request(c.Request.Method, embyHost+u.String(), c.Request.Header, c.Request.Body)
	if checkErr(c, err) {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		checkErr(c, fmt.Errorf("错误的响应码: %v", resp.StatusCode))
		return
	}

	c.Status(resp.StatusCode)
	https.CloneHeader(c, resp.Header)
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
	// 检查用户是否启用了转码版本获取
	if !config.C.VideoPreview.Enable {
		ProxyOrigin(c)
		return
	}

	// 代理请求
	embyHost := config.C.Emby.Host
	c.Request.Header.Del("Accept-Encoding")
	resp, err := https.Request(c.Request.Method, embyHost+c.Request.URL.String(), c.Request.Header, c.Request.Body)
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
		https.CloneHeader(c, resp.Header)
		jsons.OkResp(c, resJson)
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
			ms.Put("Name", jsons.NewByVal("(原画) "+originName))

			for _, tplId := range allTplIds {
				copyMs := jsons.NewByVal(ms.Struct())
				copyMs.Put("Name", jsons.NewByVal(fmt.Sprintf("(%s) %s", tplId, originName)))
				copyMs.Put("Id", jsons.NewByVal(fmt.Sprintf("%s%s%s", originId, MediaSourceIdSegment, tplId)))
				copyMs.Put("MediaStreams", proresMediaStreams)
				toAdd = append(toAdd, copyMs)
			}
			return nil
		})

		mediaSources.Append(toAdd...)
		return nil
	})
}
