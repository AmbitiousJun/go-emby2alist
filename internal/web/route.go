package web

import (
	"go-emby2alist/internal/constant"
	"go-emby2alist/internal/service/emby"
	"go-emby2alist/internal/service/m3u8"
	"log"

	"github.com/gin-gonic/gin"
)

// rules 预定义路由拦截规则, 以及相应的处理器
//
// 每个规则为一个切片, 参数分别是: 正则表达式, 处理器
var rules [][2]interface{}

func initRulePatterns() {
	log.Println("正在初始化路由规则...")
	rules = compileRules([][2]interface{}{
		// 代理 websocket
		{constant.Reg_Socket, emby.ProxySocket()},

		// 代理 PlaybackInfo 接口
		{constant.Reg_PlaybackInfo, emby.TransferPlaybackInfo},

		// 代理 Items 接口
		{constant.Reg_UserItems, emby.LoadCacheItems},

		// 重排序剧集
		{constant.Reg_ShowEpisodes, emby.ResortEpisodes},

		// 字幕长时间缓存
		{constant.Reg_VideoSubtitles, emby.ProxySubtitles},

		// 资源重定向到直链
		{constant.Reg_ResourceStream, emby.Redirect2AlistLink},
		// 代理 m3u8 转码播放列表
		{constant.Reg_ProxyPlaylist, m3u8.ProxyPlaylist},
		// 代理 ts 重定向到直链
		{constant.Reg_ProxyTs, m3u8.ProxyTsLink},
		// 代理 m3u8 字幕
		{constant.Reg_ProxySubtitle, m3u8.ProxySubtitle},

		// 资源下载, 重定向到直链
		{constant.Reg_ItemDownload, emby.Redirect2AlistLink},

		// 特定资源走代理
		//
		// ^/$: 根路径不允许重定向
		//
		// (?i): 忽略大小写
		// 有些请求使用重定向会导致部分客户端无法正常使用, 这里统一进行拦截
		{constant.Reg_Proxy2Origin, emby.ProxyOrigin},

		// 其余资源走重定向回源
		{constant.Reg_All, emby.RedirectOrigin},
	})
	log.Println("路由规则初始化完成")
}

// initRoutes 初始化路由
func initRoutes(r *gin.Engine) {
	r.Any("/*vars", globalDftHandler)
}
