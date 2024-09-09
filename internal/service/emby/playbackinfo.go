package emby

import (
	"errors"
	"fmt"
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/util/color"
	"go-emby2alist/internal/util/https"
	"go-emby2alist/internal/util/jsons"
	"go-emby2alist/internal/util/urls"
	"go-emby2alist/internal/web/cache"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (

	// PlaybackCacheSpace PlaybackInfo 的缓存空间 key
	PlaybackCacheSpace = "PlaybackInfo"
)

// PlaybackInfoReverseProxy 代理 PlaybackInfo 接口, 防止客户端转码
func PlaybackInfoReverseProxy(c *gin.Context) {
	// 1 解析资源信息
	itemInfo, err := resolveItemInfo(c)
	log.Printf(color.ToBlue("ItemInfo 解析结果: %s"), jsons.NewByVal(itemInfo))
	if checkErr(c, err) {
		return
	}

	defer func() {
		// 缓存 12h
		c.Header(cache.HeaderKeyExpired, cache.Duration(time.Hour*12))
		// 将请求结果缓存到指定缓存空间下
		c.Header(cache.HeaderKeySpace, PlaybackCacheSpace)
		c.Header(cache.HeaderKeySpaceKey, itemInfo.Id+itemInfo.MsInfo.RawId)
	}()

	msInfo := itemInfo.MsInfo
	useTranscode := !msInfo.Empty && msInfo.Transcode
	if useTranscode {
		q := c.Request.URL.Query()
		q.Set("MediaSourceId", msInfo.OriginId)
		c.Request.URL.RawQuery = q.Encode()
	}

	// 2 请求 emby 源服务器的 PlaybackInfo 信息
	res, respHeader := RawFetch(unProxyPlaybackInfoUri(c.Request.URL.String()), c.Request.Method, c.Request.Body)
	if res.Code != http.StatusOK {
		checkErr(c, errors.New(res.Msg))
		return
	}

	// 3 处理 JSON 响应
	resJson := res.Data
	mediaSources, ok := resJson.Attr("MediaSources").Done()
	if !ok || mediaSources.Type() != jsons.JsonTypeArr {
		checkErr(c, errors.New("获取不到 MediaSources 属性"))
	}

	if mediaSources.Empty() {
		log.Println(color.ToYellow("没有找到可播放的资源"))
		c.JSON(res.Code, resJson.Struct())
		return
	}

	log.Printf(color.ToBlue("获取到的 MediaSources 个数: %d"), mediaSources.Len())
	var haveReturned = errors.New("have returned")
	resChans := make([]chan []*jsons.Item, 0)
	err = mediaSources.RangeArr(func(_ int, source *jsons.Item) error {
		if !msInfo.Empty {
			// 如果客户端请求携带了 MediaSourceId 参数
			// 在返回数据时, 需要重新设置回原始的 Id
			source.Attr("Id").Set(msInfo.RawId)
		}

		if ir, ok := source.Attr("IsRemote").Bool(); ok && ir {
			// 不阻塞远程资源
			https.CloneHeader(c, respHeader)
			c.JSON(res.Code, resJson.Struct())
			return haveReturned
		}

		// 转换直链链接
		source.Attr("SupportsDirectPlay").Set(true)
		source.Attr("SupportsDirectStream").Set(true)
		newUrl := fmt.Sprintf(
			"/videos/%s/stream?MediaSourceId=%s&api_key=%s&Static=true",
			itemInfo.Id, source.Attr("Id").Val(), config.C.Emby.ApiKey,
		)

		// 简化资源名称
		name := findMediaSourceName(source)
		if name != "" {
			source.Attr("Name").Set(name)
		}
		name = source.Attr("Name").Val().(string)
		source.Attr("Name").Set(fmt.Sprintf("(原画) %s", name))

		if useTranscode {
			// 客户端请求指定的转码资源
			source.Attr("Name").Set(fmt.Sprintf("(%s) %s", msInfo.SourceNamePrefix, source.Attr("Name").Val()))
			source.Put("SupportsTranscoding", jsons.NewByVal(true))
			source.Put("TranscodingUrl", jsons.NewByVal(newUrl))
			source.Put("TranscodingSubProtocol", jsons.NewByVal("hls"))
			source.Put("TranscodingContainer", jsons.NewByVal("ts"))
			source.Put("SupportsDirectPlay", jsons.NewByVal(false))
			source.Put("SupportsDirectStream", jsons.NewByVal(false))
			log.Printf(color.ToBlue("设置转码播放链接为: %s"), newUrl)
			source.DelKey("DirectStreamUrl")
			return nil
		}
		source.Attr("DirectStreamUrl").Set(newUrl)
		log.Printf(color.ToBlue("设置直链播放链接为: %s"), newUrl)

		source.Put("SupportsTranscoding", jsons.NewByVal(false))
		source.DelKey("TranscodingUrl")
		source.DelKey("TranscodingSubProtocol")
		source.DelKey("TranscodingContainer")
		log.Println(color.ToBlue("转码配置被移除"))

		// 添加转码 MediaSource 获取
		cfg := config.C.VideoPreview
		if !msInfo.Empty || !cfg.Enable || !cfg.ContainerValid(source.Attr("Container").Val().(string)) {
			return nil
		}
		resChan := make(chan []*jsons.Item, 1)
		go findVideoPreviewInfos(source, name, resChan)
		resChans = append(resChans, resChan)
		return nil
	})

	if err == haveReturned {
		return
	}

	// 收集异步请求的转码资源信息
	for _, resChan := range resChans {
		previewInfos := <-resChan
		if len(previewInfos) > 0 {
			log.Printf(color.ToGreen("找到 %d 个转码资源信息"), len(previewInfos))
			mediaSources.Append(previewInfos...)
		}
	}

	respHeader.Del("Content-Length")
	https.CloneHeader(c, respHeader)
	c.JSON(res.Code, resJson.Struct())
}

// TransferPlaybackInfo 拦截并代理 PlaybackInfo
//
// 如果 PlaybackInfo 缓存空间有相应的缓存
// 则直接使用缓存响应
func TransferPlaybackInfo(c *gin.Context) {
	// 1 解析出 ItemId
	itemInfo, err := resolveItemInfo(c)
	if err != nil {
		return
	}
	log.Printf(color.ToBlue("itemInfo 解析结果: %s"), jsons.NewByVal(itemInfo))

	// 2 从缓存空间中获取数据
	if res, ok := getPlaybackInfoByCacheSpace(itemInfo); ok {
		log.Println(color.ToBlue("使用缓存空间中的 PlaybackInfo 响应"))
		c.JSON(http.StatusOK, res.Struct())
		return
	}

	// 3 请求代理接口
	checkErr(c, https.ProxyRequest(c, https.ClientRequestHost(c)+itemInfo.PlaybackInfoRpUri, false))
}

// LoadCacheItems 拦截并代理 items 接口
//
// 如果 PlaybackInfo 缓存空间有相应的缓存
// 则将缓存中的 MediaSources 信息覆盖到响应中
//
// 防止转码资源信息丢失
func LoadCacheItems(c *gin.Context) {
	// 1 代理请求
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

	// 2 未开启转码资源获取功能
	if !config.C.VideoPreview.Enable {
		return
	}

	// 3 解析出 ItemId
	itemInfo, err := resolveItemInfo(c)
	if err != nil {
		return
	}
	log.Printf(color.ToBlue("itemInfo 解析结果: %s"), jsons.NewByVal(itemInfo))

	// 4 获取附带转码信息的 PlaybackInfo 数据
	var cacheBody *jsons.Item
	if cb, ok := getPlaybackInfoByCacheSpace(itemInfo); ok {
		cacheBody = cb
		log.Println(color.ToBlue("使用缓存空间中的 MediaSources 覆盖原始响应"))
	} else if cb, ok := getPlaybackInfoByRequest(c, itemInfo); ok {
		cacheBody = cb
		log.Println(color.ToBlue("请求最新的 MediaSources 来覆盖原始响应"))
	}
	if cacheBody == nil {
		return
	}

	cacheMs, ok := cacheBody.Attr("MediaSources").Done()
	if !ok || cacheMs.Type() != jsons.JsonTypeArr {
		return
	}

	// 5 覆盖原始响应
	resJson.Put("MediaSources", cacheMs)
	c.Writer.Header().Del("Content-Length")
}

// getPlaybackInfoByRequest 请求本地的 PlaybackInfo 接口获取信息
func getPlaybackInfoByRequest(c *gin.Context, itemInfo ItemInfo) (*jsons.Item, bool) {
	if c == nil {
		return nil, false
	}
	url := https.ClientRequestHost(c) + itemInfo.PlaybackInfoRpUri
	url = urls.AppendArgs(url, "ignore_error", "true")
	resp, err := https.Request(http.MethodPost, url, nil, nil)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false
	}
	resJson, err := jsons.New(string(bodyBytes))
	if err != nil {
		return nil, false
	}
	return resJson, true
}

// getPlaybackInfoByCacheSpace 从缓存空间中获取 PlaybackInfo 信息
func getPlaybackInfoByCacheSpace(itemInfo ItemInfo) (*jsons.Item, bool) {
	spaceCache, ok := cache.GetSpaceCache(PlaybackCacheSpace, itemInfo.Id+itemInfo.MsInfo.RawId)
	if !ok {
		return nil, false
	}

	// 5 获取缓存响应体内容
	cacheBody, err := spaceCache.JsonBody()
	if err != nil {
		return nil, false
	}
	return cacheBody, true
}

// proxyPlaybackInfoUri 将 PlaybackInfo uri 转成反向代理 uri
func proxyPlaybackInfoUri(uri string) string {
	return strings.ReplaceAll(uri, "/PlaybackInfo", "/PlaybackInfo/ReverseProxy")
}

// unProxyPlaybackInfoUri 去除 PlaybackInfo 反向代理 uri
func unProxyPlaybackInfoUri(pxyUri string) string {
	return strings.ReplaceAll(pxyUri, "/PlaybackInfo/ReverseProxy", "/PlaybackInfo")
}
