package constant

const (
	CurrentVersion = "v1.5.1-beta-v3"
	RepoAddr       = "https://github.com/AmbitiousJun/go-emby2alist"
)

const (
	Reg_Socket                   = `(?i)^/.*(socket|embywebsocket)`
	Reg_PlaybackInfo             = `(?i)^/.*items/.*/playbackinfo\??`
	Reg_PlayingStopped           = `(?i)^/.*sessions/playing/stopped`
	Reg_PlayingProgress          = `(?i)^/.*sessions/playing/progress`
	Reg_UserItems                = `(?i)^/.*users/.*/items/\d+($|\?)`
	Reg_UserEpisodeItems         = `(?i)^/.*users/.*/items\?.*includeitemtypes=(episode|movie)`
	Reg_UserItemsRandomResort    = `(?i)^/.*users/.*/items\?.*SortBy=Random`
	Reg_UserItemsRandomWithLimit = `(?i)^/.*users/.*/items/with_limit\?.*SortBy=Random`
	Reg_ShowEpisodes             = `(?i)^/.*shows/.*/episodes\??`
	Reg_VideoSubtitles           = `(?i)^/.*videos/.*/subtitles`
	Reg_ResourceStream           = `(?i)^/.*(videos|audio)/.*/(stream|universal)(\.\w+)?\??`
	Reg_ResourceMaster           = `(?i)^/.*(videos|audio)/.*/(master)(\.\w+)?\??`
	Reg_ResourceMain             = `(?i)^/.*(videos|audio)/.*/main.m3u8\??`
	Reg_ProxyPlaylist            = `(?i)^/.*videos/proxy_playlist\??`
	Reg_ProxyTs                  = `(?i)^/.*videos/proxy_ts\??`
	Reg_ProxySubtitle            = `(?i)^/.*videos/proxy_subtitle\??`
	Reg_ItemDownload             = `(?i)^/.*items/\d+/download($|\?)`
	Reg_ItemSyncDownload         = `(?i)^/.*sync/jobitems/\d+/file($|\?)`
	Reg_Images                   = `(?i)^/.*images`
	Reg_VideoModWebDefined       = `(?i)^/web/modules/htmlvideoplayer/plugin.js`
	Reg_Proxy2Origin             = `^/$|(?i)^.*(/web|/users|/artists|/genres|/similar|/shows|/system|/remote|/scheduledtasks)`
	Reg_All                      = `.*`
)
