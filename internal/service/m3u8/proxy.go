package m3u8

import (
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/util/color"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ProxyPlaylist 代理 m3u8 转码地址
func ProxyPlaylist(c *gin.Context) {
	if c.Request.Method != http.MethodGet {
		c.String(http.StatusMethodNotAllowed, "仅支持 GET")
		return
	}

	alistPath, err := url.QueryUnescape(strings.TrimSpace(c.Query("alist_path")))
	if err != nil {
		c.String(http.StatusBadRequest, "alistPath 转换失败: %v", err)
		return
	}
	templateId := strings.TrimSpace(c.Query("template_id"))
	apiKey := strings.TrimSpace(c.Query("api_key"))
	remote := strings.TrimSpace(c.Query("remote"))
	if alistPath == "" || templateId == "" || apiKey == "" {
		c.String(http.StatusBadRequest, "参数不足")
		return
	}

	if apiKey != config.C.Emby.ApiKey {
		c.String(http.StatusUnauthorized, "无权限访问")
		return
	}

	okContent := func(content string) {
		c.Header("Content-Type", "application/vnd.apple.mpegurl")
		c.String(http.StatusOK, content)
	}

	m3uContent, ok := GetPlaylist(alistPath, templateId, true)
	if ok {
		okContent(m3uContent)
		return
	}

	// 获取失败, 将当前请求的地址加入到预处理通道
	PushPlaylistAsync(Info{AlistPath: alistPath, TemplateId: templateId, Remote: remote})

	// 重新获取一次
	m3uContent, ok = GetPlaylist(alistPath, templateId, true)
	if ok {
		okContent(m3uContent)
		return
	}
	c.String(http.StatusBadRequest, "获取不到播放列表, 请检查日志")
}

// ProxyTsLink 代理 ts 直链地址
func ProxyTsLink(c *gin.Context) {
	if c.Request.Method != http.MethodGet {
		c.String(http.StatusMethodNotAllowed, "仅支持 GET")
		return
	}

	alistPath, err := url.QueryUnescape(strings.TrimSpace(c.Query("alist_path")))
	if err != nil {
		c.String(http.StatusBadRequest, "alistPath 转换失败: %v", err)
		return
	}
	idxStr := strings.TrimSpace(c.Query("idx"))
	templateId := strings.TrimSpace(c.Query("template_id"))
	apiKey := strings.TrimSpace(c.Query("api_key"))
	if alistPath == "" || idxStr == "" || templateId == "" || apiKey == "" {
		c.String(http.StatusBadRequest, "参数不足")
		return
	}
	if config.C.Emby.ApiKey != apiKey {
		c.String(http.StatusUnauthorized, "无权限访问")
		return
	}
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 {
		c.String(http.StatusBadRequest, "无效 idx")
		return
	}

	okRedirect := func(link string) {
		log.Printf(color.ToGreen("重定向 ts: %s"), link)
		c.Redirect(http.StatusTemporaryRedirect, link)
	}

	tsLink, ok := GetTsLink(alistPath, templateId, idx)
	if ok {
		okRedirect(tsLink)
		return
	}

	// 获取失败, 将当前请求的地址加入到预处理通道
	PushPlaylistAsync(Info{AlistPath: alistPath, TemplateId: templateId})

	tsLink, ok = GetTsLink(alistPath, templateId, idx)
	if ok {
		okRedirect(tsLink)
		return
	}
	c.String(http.StatusBadRequest, "获取不到 ts, 请检查日志")
}
