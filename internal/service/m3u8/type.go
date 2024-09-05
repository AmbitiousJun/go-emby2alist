package m3u8

// Info 记录一个 m3u8 相关信息
type Info struct {
	AlistPath    string   // 资源在 alist 中的绝对路径
	TemplateId   string   // 转码资源模板 id
	RemoteBase   string   // 远程 m3u8 地址前缀
	RemoteTsUrls []string // 远程的 ts URL 列表, 用于重定向

	// LastRead 客户端最后读取的时间戳 (毫秒)
	//
	// 超过 30 分钟未读取, 程序停止更新;
	// 超过 12 小时未读取, m3u info 被移除
	LastRead int64

	// LastUpdate 程序最后的更新时间戳 (毫秒)
	//
	// 客户端来读取时, 如果 m3u info 已经超过 10 分钟没有更新了
	// 触发更新机制之后, 再返回最新的地址
	LastUpdate int64
}
