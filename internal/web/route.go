package web

import (
	"go-emby2alist/internal/service/emby"
	"log"

	"github.com/gin-gonic/gin"
)

// rules 预定义路由拦截规则, 以及相应的处理器
//
// 每个规则为一个切片, 参数分别是: 正则表达式, 处理器
var rules [][2]interface{}

func init() {
	log.Println("正在初始化路由规则...")
	rules = compileRules([][2]interface{}{
		// 代理 websocket
		{`(?i)^/.*(socket|embywebsocket)`, emby.ProxySocket()},

		// 代理 PlaybackInfo 接口
		{`(?i)^/.*items/.*/playbackinfo\??`, emby.TransferPlaybackInfo},

		// 代理 Items 接口
		{`(?i)^/.*users/.*/items/\d+($|\?)`, emby.LoadCacheItems},

		// 资源重定向到直链
		{`(?i)^/.*(videos|audio)/.*/(stream|universal)\??`, emby.Redirect2AlistLink},

		// 资源下载, 重定向到直链
		{`(?i)^/.*items/.*/download`, emby.Redirect2AlistLink},

		// 字幕长时间缓存
		{`(?i)^/.*videos/.*/subtitles`, emby.ProxySubtitles},

		// 特定资源走代理
		//
		// ^/$: 根路径不允许重定向
		//
		// (?i): 忽略大小写
		// 有些请求使用重定向会导致部分客户端无法正常使用, 这里统一进行拦截
		{`^/$|(?i)^/.*(web|users|artists|genres|similar|shows|system|remote|scheduledtasks)`, emby.ProxyOrigin},

		// 其余资源走重定向回源
		{`.*`, emby.RedirectOrigin},
	})
	log.Println("路由规则初始化完成")
}

// initRoutes 初始化路由
func initRoutes(r *gin.Engine) {
	r.Any("/*vars", globalDftHandler)
}
