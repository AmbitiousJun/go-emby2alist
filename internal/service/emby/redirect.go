package emby

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/service/alist"
	"github.com/AmbitiousJun/go-emby2alist/internal/service/path"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/https"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/jsons"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/strs"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/urls"
	"github.com/AmbitiousJun/go-emby2alist/internal/web/cache"

	"github.com/gin-gonic/gin"
)

// Redirect2Transcode 将 master 请求重定向到本地 ts 代理
func Redirect2Transcode(c *gin.Context) {
	// 只有三个必要的参数都获取到时, 才跳转到本地 ts 代理
	templateId := c.Query("template_id")
	apiKey := c.Query(QueryApiKeyName)
	alistPath := c.Query("alist_path")
	if strs.AnyEmpty(templateId, apiKey, alistPath) {
		ProxyOrigin(c)
		return
	}
	log.Println(colors.ToBlue("检测到自定义的转码 m3u8 请求, 重定向到本地代理接口"))
	tu, _ := url.Parse("/videos/proxy_playlist")
	q := tu.Query()
	q.Set("alist_path", alistPath)
	q.Set(QueryApiKeyName, apiKey)
	q.Set("template_id", templateId)
	tu.RawQuery = q.Encode()
	c.Redirect(http.StatusTemporaryRedirect, tu.String())
}

// Redirect2AlistLink 重定向资源到 alist 网盘直链
func Redirect2AlistLink(c *gin.Context) {
	// 1 解析要请求的资源信息
	itemInfo, err := resolveItemInfo(c)
	if checkErr(c, err) {
		return
	}
	log.Printf(colors.ToBlue("解析到的 itemInfo: %v"), jsons.NewByVal(itemInfo))

	// 2 如果请求的是转码资源, 重定向到本地的 m3u8 代理服务
	msInfo := itemInfo.MsInfo
	useTranscode := !msInfo.Empty && msInfo.Transcode
	if useTranscode && msInfo.AlistPath != "" {
		u, _ := url.Parse(strings.ReplaceAll(MasterM3U8UrlTemplate, "${itemId}", itemInfo.Id))
		q := u.Query()
		q.Set("template_id", itemInfo.MsInfo.TemplateId)
		q.Set(QueryApiKeyName, config.C.Emby.ApiKey)
		q.Set("alist_path", itemInfo.MsInfo.AlistPath)
		u.RawQuery = q.Encode()
		log.Printf(colors.ToGreen("重定向 playlist: %s"), u.String())
		c.Redirect(http.StatusTemporaryRedirect, u.String())
		return
	}

	// 3 请求资源在 Emby 中的 Path 参数
	embyPath, err := getEmbyFileLocalPath(itemInfo)
	if checkErr(c, err) {
		return
	}

	// 4 如果是远程地址 (strm), 直接进行重定向
	if urls.IsRemote(embyPath) {
		finalPath := config.C.Emby.Strm.MapPath(embyPath)
		log.Printf(colors.ToGreen("重定向 strm: %s"), finalPath)
		c.Header(cache.HeaderKeyExpired, "-1")
		c.Redirect(http.StatusTemporaryRedirect, finalPath)
		return
	}

	// 5 请求 alist 资源
	fi := alist.FetchInfo{
		Header:       c.Request.Header.Clone(),
		UseTranscode: useTranscode,
		Format:       msInfo.TemplateId,
	}
	alistPathRes := path.Emby2Alist(embyPath)

	allErrors := strings.Builder{}
	// handleAlistResource 根据传递的 path 请求 alist 资源
	handleAlistResource := func(path string) bool {
		log.Printf(colors.ToBlue("尝试请求 Alist 资源: %s"), path)
		fi.Path = path
		res := alist.FetchResource(fi)

		if res.Code != http.StatusOK {
			allErrors.WriteString(fmt.Sprintf("请求 Alist 失败, code: %d, msg: %s, path: %s;", res.Code, res.Msg, path))
			return false
		}

		// 处理直链
		if !fi.UseTranscode {
			log.Printf(colors.ToGreen("请求成功, 重定向到: %s"), res.Data.Url)
			c.Header(cache.HeaderKeyExpired, cache.Duration(time.Minute*10))
			c.Redirect(http.StatusTemporaryRedirect, res.Data.Url)
			return true
		}

		// 代理转码 m3u
		u, _ := url.Parse(strings.ReplaceAll(https.ClientRequestHost(c)+MasterM3U8UrlTemplate, "${itemId}", itemInfo.Id))
		q := u.Query()
		q.Set("template_id", itemInfo.MsInfo.TemplateId)
		q.Set(QueryApiKeyName, config.C.Emby.ApiKey)
		q.Set("alist_path", path)
		u.RawQuery = q.Encode()
		_, resp, err := https.RequestRedirect(http.MethodGet, u.String(), nil, nil, true)
		if err != nil {
			allErrors.WriteString(fmt.Sprintf("代理转码 m3u 失败: %v;", err))
			return false
		}
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if checkErr(c, err) {
			return true
		}
		c.Status(resp.StatusCode)
		https.CloneHeader(c, resp.Header)
		c.Writer.Write(bodyBytes)
		c.Writer.Flush()
		return true
	}

	if alistPathRes.Success && handleAlistResource(alistPathRes.Path) {
		return
	}
	paths, err := alistPathRes.Range()
	if checkErr(c, err) {
		return
	}
	for _, path := range paths {
		if handleAlistResource(path) {
			return
		}
	}

	checkErr(c, fmt.Errorf("获取直链失败: %s", allErrors.String()))
}

// checkErr 检查 err 是否为空
// 不为空则根据错误处理策略返回响应
//
// 返回 true 表示请求已经被处理
//
// 如果检测到 query 参数 ignore_error 为 true, 则不进行重定向
func checkErr(c *gin.Context, err error) bool {
	if err == nil || c == nil {
		return false
	}

	// 异常接口, 不缓存
	c.Header(cache.HeaderKeyExpired, "-1")

	// 请求参数中有忽略异常
	if c.Query("ignore_error") == "true" {
		c.String(http.StatusOK, "error has been ignored")
		return true
	}

	// 采用拒绝策略, 直接返回错误
	if config.C.Emby.ProxyErrorStrategy == config.StrategyReject {
		log.Printf(colors.ToRed("代理接口失败: %v"), err)
		c.String(http.StatusInternalServerError, "代理接口失败, 请检查日志")
		return true
	}

	u := config.C.Emby.Host + c.Request.URL.String()
	log.Printf(colors.ToRed("代理接口失败: %v, 重定向回源服务器处理\n"), err)
	c.Redirect(http.StatusTemporaryRedirect, u)
	return true
}
