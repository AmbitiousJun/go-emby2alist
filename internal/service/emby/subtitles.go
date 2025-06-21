package emby

import (
	"net/http"
	"net/url"
	"time"

	"github.com/AmbitiousJun/go-emby2openlist/internal/util/strs"
	"github.com/AmbitiousJun/go-emby2openlist/internal/web/cache"
	"github.com/gin-gonic/gin"
)

// ProxySubtitles 字幕代理, 过期时间设置为 30 天
func ProxySubtitles(c *gin.Context) {
	if c == nil {
		return
	}

	// 判断是否带有转码字幕参数
	openlistPath := c.Query("openlist_path")
	templateId := c.Query("template_id")
	subName := c.Query("sub_name")
	apiKey := c.Query(QueryApiKeyName)
	if strs.AllNotEmpty(openlistPath, templateId, subName, apiKey) {
		u, _ := url.Parse("/videos/proxy_subtitle")
		u.RawQuery = c.Request.URL.RawQuery
		c.Redirect(http.StatusTemporaryRedirect, u.String())
		return
	}

	c.Header(cache.HeaderKeyExpired, cache.Duration(time.Hour*24*30))
	ProxyOrigin(c)
}
