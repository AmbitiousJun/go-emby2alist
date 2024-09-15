package constant

const (
	Reg_Socket         = `(?i)^/.*(socket|embywebsocket)`
	Reg_PlaybackInfo   = `(?i)^/.*items/.*/playbackinfo\??`
	Reg_UserItems      = `(?i)^/.*users/.*/items/\d+($|\?)`
	Reg_ShowEpisodes   = `(?i)^/.*shows/.*/episodes\??`
	Reg_VideoSubtitles = `(?i)^/.*videos/.*/subtitles`
	Reg_ResourceStream = `(?i)^/.*(videos|audio)/.*/(stream|universal)(\.\w+)?\??`
	Reg_ProxyPlaylist  = `(?i)^/.*videos/proxy_playlist\??`
	Reg_ProxyTs        = `(?i)^/.*videos/proxy_ts\??`
	Reg_ProxySubtitle  = `(?i)^/.*videos/proxy_subtitle\??`
	Reg_ItemDownload   = `(?i)^/.*items/.*/download`
	Reg_Proxy2Origin   = `^/$|(?i)^.*(/web|/users|/artists|/genres|/similar|/shows|/system|/remote|/scheduledtasks)`
	Reg_All            = `.*`
)
