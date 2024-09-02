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
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (

	// PlaybackCacheSpace PlaybackInfo 的缓存空间 key
	PlaybackCacheSpace = "PlaybackInfo"
)

// TransferPlaybackInfo 代理 PlaybackInfo 接口, 防止客户端转码
func TransferPlaybackInfo(c *gin.Context) {
	// 1 解析资源信息
	itemInfo, err := resolveItemInfo(c)
	log.Printf(color.ToBlue("ItemInfo 解析结果: %s"), jsons.NewByVal(itemInfo))
	if checkErr(c, err) {
		return
	}

	defer func() {
		// 将请求结果缓存到指定缓存空间下
		if !itemInfo.MsInfo.Empty {
			// 请求指定资源的 PlaybackInfo 不缓存
			return
		}
		c.Header(cache.HeaderKeySpace, PlaybackCacheSpace)
		c.Header(cache.HeaderKeySpaceKey, itemInfo.Id)
	}()

	msInfo := itemInfo.MsInfo
	useTranscode := !msInfo.Empty && msInfo.Transcode
	if useTranscode {
		q := c.Request.URL.Query()
		q.Set("MediaSourceId", msInfo.OriginId)
		c.Request.URL.RawQuery = q.Encode()
	}

	// 2 请求 emby 源服务器的 PlaybackInfo 信息
	res, respHeader := RawFetch(c.Request.URL.String(), c.Request.Method, c.Request.Body)
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

	log.Printf(color.ToBlue("获取到的 MediaSources 个数: %s"), mediaSources.Len())
	var haveReturned = errors.New("have returned")
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
		newUrl := urls.ReplaceAll(
			c.Request.URL.String(),
			"/emby/Items", "/videos",
			"PlaybackInfo", "stream",
		)
		newUrl = urls.AppendArgs(
			newUrl,
			"MediaSourceId", source.Attr("Id").Val().(string),
			"Static", "true",
		)

		// 简化资源名称
		name := findMediaSourceName(source)
		if name != "" {
			source.Attr("Name").Set(name)
		}

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
		previewInfos := findVideoPreviewInfos(source)
		if len(previewInfos) > 0 {
			log.Printf(color.ToGreen("找到 %d 个转码资源信息"), len(previewInfos))
			mediaSources.Append(previewInfos...)
		}

		source.Attr("Name").Set(fmt.Sprintf("(原画) %s", source.Attr("Name").Val()))
		return nil
	})

	if err == haveReturned {
		return
	}

	respHeader.Del("Content-Length")
	https.CloneHeader(c, respHeader)
	c.JSON(res.Code, resJson.Struct())
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

	// 2 解析出 ItemId
	itemInfo, err := resolveItemInfo(c)
	if err != nil {
		return
	}
	log.Printf(color.ToBlue("itemInfo 解析结果: %s"), jsons.NewByVal(itemInfo))

	// 3 查询缓存空间的 PlaybackInfo 缓存
	cache, ok := cache.GetSpaceCache(PlaybackCacheSpace, itemInfo.Id)
	if !ok {
		return
	}

	// 4 获取缓存响应体内容
	cacheBody, err := cache.JsonBody()
	if err != nil {
		return
	}
	cacheMs, ok := cacheBody.Attr("MediaSources").Done()
	if !ok || cacheMs.Type() != jsons.JsonTypeArr {
		return
	}
	log.Println(color.ToBlue("使用缓存空间中的 MediaSources 覆盖原始响应"))

	// 5 覆盖原始响应
	resJson.Put("MediaSources", cacheMs)
	c.Writer.Header().Del("Content-Length")
}
