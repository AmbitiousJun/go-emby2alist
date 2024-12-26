package web

import (
	"log"

	"github.com/AmbitiousJun/go-emby2alist/internal/constant"
	"github.com/AmbitiousJun/go-emby2alist/internal/service/emby"
	"github.com/AmbitiousJun/go-emby2alist/internal/service/m3u8"

	"github.com/gin-gonic/gin"
)

// rules 预定义路由拦截规则, 以及相应的处理器
//
// 每个规则为一个切片, 参数分别是: 正则表达式, 处理器
var rules [][2]interface{}

func initRulePatterns() {
	log.Println("正在初始化路由规则...")
	rules = compileRules([][2]interface{}{
		// websocket
		{constant.Reg_Socket, emby.ProxySocket()},

		// PlaybackInfo 接口
		{constant.Reg_PlaybackInfo, emby.TransferPlaybackInfo},

		// Items 接口
		{constant.Reg_UserItems, emby.LoadCacheItems},
		// 代理 Items 并添加转码版本信息
		{constant.Reg_UserEpisodeItems, emby.ProxyAddItemsPreviewInfo},
		// 随机列表接口
		{constant.Reg_UserItemsRandomResort, emby.ResortRandomItems},
		// 代理原始的随机列表接口, 去除 limit 限制, 并进行缓存
		{constant.Reg_UserItemsRandomWithLimit, emby.RandomItemsWithLimit},

		// 重排序剧集
		{constant.Reg_ShowEpisodes, emby.ResortEpisodes},

		// 字幕长时间缓存
		{constant.Reg_VideoSubtitles, emby.ProxySubtitles},

		// 资源重定向到直链
		{constant.Reg_ResourceStream, emby.Redirect2AlistLink},
		// master 重定向到本地 m3u8 代理
		{constant.Reg_ResourceMaster, emby.Redirect2Transcode},
		// main 路由到直链接口
		{constant.Reg_ResourceMain, emby.Redirect2AlistLink},
		// m3u8 转码播放列表
		{constant.Reg_ProxyPlaylist, m3u8.ProxyPlaylist},
		// ts 重定向到直链
		{constant.Reg_ProxyTs, m3u8.ProxyTsLink},
		// m3u8 字幕
		{constant.Reg_ProxySubtitle, m3u8.ProxySubtitle},
		// 资源下载, 重定向到直链
		{constant.Reg_ItemDownload, emby.Redirect2AlistLink},

		// 处理图片请求
		{constant.Reg_Images, emby.HandleImages},

		// web cors 处理
		{constant.Reg_BaseVideoModWebDefined, emby.ChangeBaseVideoModuleCorsDefined},

		// 其余资源走重定向回源
		{constant.Reg_All, emby.ProxyOrigin},
	})
	log.Println("路由规则初始化完成")
}

// initRoutes 初始化路由
func initRoutes(r *gin.Engine) {
	r.Any("/*vars", globalDftHandler)
}
