package emby

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/service/alist"
	"github.com/AmbitiousJun/go-emby2alist/internal/service/path"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/https"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/jsons"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/strs"
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
		// 重定向回源
		checkErr(c, fmt.Errorf("获取不到核心参数, templateId: %s, apiKey: %s, alistPath: %s", templateId, apiKey, alistPath))
		return
	}
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
	if useTranscode {
		u, _ := url.Parse("/videos/proxy_playlist")
		q := u.Query()
		q.Set("template_id", itemInfo.MsInfo.TemplateId)
		q.Set(QueryApiKeyName, config.C.Emby.ApiKey)
		q.Set("alist_path", itemInfo.MsInfo.AlistPath)
		u.RawQuery = q.Encode()
		res := https.ClientRequestHost(c) + u.String()
		log.Printf(colors.ToGreen("重定向 playlist: %s"), res)
		c.Redirect(http.StatusTemporaryRedirect, res)
		return
	}

	// 3 请求资源在 Emby 中的 Path 参数
	embyPath, err := getEmbyFileLocalPath(itemInfo)
	if checkErr(c, err) {
		return
	}

	// 4 请求 alist 资源
	fi := alist.FetchInfo{}
	fi.Header = c.Request.Header.Clone()
	alistPathRes := path.Emby2Alist(embyPath)
	if alistPathRes.Success {
		log.Printf(colors.ToBlue("尝试请求 Alist 资源: %s"), alistPathRes.Path)
		fi.Path = alistPathRes.Path
		res := alist.FetchResource(fi)

		if res.Code == http.StatusOK {
			log.Printf(colors.ToGreen("请求成功, 重定向到: %s"), res.Data.Url)
			c.Header(cache.HeaderKeyExpired, cache.Duration(time.Minute*10))
			c.Redirect(http.StatusTemporaryRedirect, res.Data.Url)
			return
		}

		if res.Code == http.StatusForbidden {
			log.Printf(colors.ToRed("请求 Alist 被阻止: %s"), res.Msg)
			c.String(http.StatusForbidden, res.Msg)
		}
	}

	paths, err := alistPathRes.Range()
	if checkErr(c, err) {
		return
	}

	for _, path := range paths {
		log.Printf(colors.ToBlue("尝试请求 Alist 资源: %s"), path)
		fi.Path = path
		res := alist.FetchResource(fi)

		if res.Code == http.StatusOK {
			log.Printf(colors.ToGreen("请求成功, 重定向到: %s"), res.Data.Url)
			c.Header(cache.HeaderKeyExpired, cache.Duration(time.Minute*10))
			c.Redirect(http.StatusTemporaryRedirect, res.Data.Url)
			return
		}
	}

	checkErr(c, errors.New("查无 Alist 直链资源"))
}

// checkErr 检查 err 是否为空
// 不为空则重定向到源服务器
//
// 返回 true 表示已重定向
//
// 如果检测到 query 参数 ignore_error 为 true, 则不进行重定向
func checkErr(c *gin.Context, err error) bool {
	if err == nil || c == nil {
		return false
	}

	// 异常接口, 不缓存
	c.Header(cache.HeaderKeyExpired, "-1")

	if c.Query("ignore_error") == "true" {
		c.String(http.StatusOK, "error has been ignored: "+err.Error())
		return true
	}

	u := config.C.Emby.Host + c.Request.URL.String()
	log.Printf(colors.ToRed("代理接口失败: %v, 重定向回源服务器处理\n"), err)
	c.Redirect(http.StatusTemporaryRedirect, u)
	return true
}
