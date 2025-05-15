package m3u8

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/https"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/strs"

	"github.com/gin-gonic/gin"
)

// baseCheck 对代理请求参数作基本校验
func baseCheck(c *gin.Context) (ProxyParams, error) {
	if c.Request.Method != http.MethodGet {
		return ProxyParams{}, errors.New("仅支持 GET")
	}

	var params ProxyParams
	if err := c.ShouldBindQuery(&params); err != nil {
		return ProxyParams{}, err
	}

	alistPath, err := url.QueryUnescape(strings.TrimSpace(params.AlistPath))
	if err != nil {
		return ProxyParams{}, fmt.Errorf("alistPath 转换失败: %v", err)
	}
	params.AlistPath = alistPath

	if params.AlistPath == "" || params.TemplateId == "" || params.ApiKey == "" {
		return ProxyParams{}, errors.New("参数不足")
	}

	return params, nil
}

// ProxyPlaylist 代理 m3u8 转码地址
func ProxyPlaylist(c *gin.Context) {
	params, err := baseCheck(c)
	if err != nil {
		log.Printf(colors.ToRed("代理 m3u8 失败: %v"), err.Error())
		c.String(http.StatusBadRequest, "代理 m3u8 失败, 请检查日志")
		return
	}

	okContent := func(content string) {
		c.Header("Content-Type", "application/vnd.apple.mpegurl")
		c.String(http.StatusOK, content)
	}

	// ts 切片使用绝对路径
	routePrefix := https.ClientRequestHost(c) + "/videos"

	m3uContent, ok := GetPlaylist(params.AlistPath, params.TemplateId, true, true, routePrefix, params.ApiKey)
	if ok {
		okContent(m3uContent)
		return
	}

	// 获取失败, 将当前请求的地址加入到预处理通道
	PushPlaylistAsync(Info{AlistPath: params.AlistPath, TemplateId: params.TemplateId})

	// 重新获取一次
	m3uContent, ok = GetPlaylist(params.AlistPath, params.TemplateId, true, true, routePrefix, params.ApiKey)
	if ok {
		okContent(m3uContent)
		return
	}
	c.String(http.StatusBadRequest, "获取不到播放列表, 请检查日志")
}

// ProxyTsLink 代理 ts 直链地址
func ProxyTsLink(c *gin.Context) {
	params, err := baseCheck(c)
	if err != nil {
		log.Printf(colors.ToRed("代理 ts 失败: %v"), err)
		c.String(http.StatusBadRequest, "代理 ts 失败, 请检查日志")
		return
	}

	idx, err := strconv.Atoi(params.IdxStr)
	if err != nil || idx < 0 {
		c.String(http.StatusBadRequest, "无效 idx")
		return
	}

	okRedirect := func(link string) {
		log.Printf(colors.ToGreen("重定向 ts: %s"), link)
		c.Redirect(http.StatusTemporaryRedirect, link)
	}

	tsLink, ok := GetTsLink(params.AlistPath, params.TemplateId, idx)
	if ok {
		okRedirect(tsLink)
		return
	}

	// 获取失败, 将当前请求的地址加入到预处理通道
	PushPlaylistAsync(Info{AlistPath: params.AlistPath, TemplateId: params.TemplateId})

	tsLink, ok = GetTsLink(params.AlistPath, params.TemplateId, idx)
	if ok {
		okRedirect(tsLink)
		return
	}
	c.String(http.StatusBadRequest, "获取不到 ts, 请检查日志")
}

// ProxySubtitle 代理字幕请求
func ProxySubtitle(c *gin.Context) {
	params, err := baseCheck(c)
	if err != nil {
		log.Printf(colors.ToRed("代理字幕失败: %v"), err)
		c.String(http.StatusBadRequest, "代理字幕失败, 请检查日志")
		return
	}

	subName := c.Query("sub_name")
	if strs.AnyEmpty(subName) {
		c.String(http.StatusBadRequest, "代理字幕失败, 缺少 sub_name 参数")
		return
	}

	proxySubtitle := func(link string) {
		log.Printf(colors.ToGreen("代理字幕: %s"), link)
		_, resp, err := https.RequestRedirect(http.MethodGet, link, nil, nil, true)
		if err != nil {
			log.Printf(colors.ToRed("代理字幕失败: %v"), err)
			c.String(http.StatusInternalServerError, "代理字幕失败, 请检查日志")
			return
		}
		defer resp.Body.Close()
		https.CloneHeader(c, resp.Header)
		c.Status(resp.StatusCode)
		if _, err = io.Copy(c.Writer, resp.Body); err != nil {
			log.Printf(colors.ToRed("代理字幕失败: %v"), err)
			c.String(http.StatusInternalServerError, "代理字幕失败, 请检查日志")
			return
		}
	}

	subtitleLink, ok := GetSubtitleLink(params.AlistPath, params.TemplateId, subName)
	if ok {
		proxySubtitle(subtitleLink)
		return
	}

	// 获取失败, 将当前请求的地址加入到预处理通道
	PushPlaylistAsync(Info{AlistPath: params.AlistPath, TemplateId: params.TemplateId})

	subtitleLink, ok = GetSubtitleLink(params.AlistPath, params.TemplateId, subName)
	if ok {
		proxySubtitle(subtitleLink)
		return
	}
	c.String(http.StatusBadRequest, "获取不到字幕")
}
