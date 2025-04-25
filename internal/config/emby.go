package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AmbitiousJun/go-emby2alist/internal/util/maps"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/randoms"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/strs"
)

// PeStrategy 代理异常策略类型
type PeStrategy string

const (
	PeStrategyOrigin PeStrategy = "origin" // 回源
	PeStrategyReject PeStrategy = "reject" // 拒绝请求
)

// DlStrategy 下载策略类型
type DlStrategy string

const (
	DlStrategyOrigin DlStrategy = "origin" // 代理到源服务器
	DlStrategyDirect DlStrategy = "direct" // 获取并重定向到直链
	DlStrategy403    DlStrategy = "403"    // 拒绝响应
)

// validPeStrategy 用于校验用户配置的策略是否合法
var validPeStrategy = map[PeStrategy]struct{}{
	PeStrategyOrigin: {}, PeStrategyReject: {},
}

// validDlStrategy 用于校验用户配置的下载策略是否合法
var validDlStrategy = map[DlStrategy]struct{}{
	DlStrategyOrigin: {}, DlStrategyDirect: {}, DlStrategy403: {},
}

// Emby 相关配置
type Emby struct {
	// Emby 源服务器地址
	Host string `yaml:"host"`
	// rclone 或者 cd 的挂载目录
	MountPath string `yaml:"mount-path"`
	// EpisodesUnplayPrior 在获取剧集列表时是否将未播资源优先展示
	EpisodesUnplayPrior bool `yaml:"episodes-unplay-prior"`
	// ResortRandomItems 是否对随机的 items 进行重排序
	ResortRandomItems bool `yaml:"resort-random-items"`
	// ProxyErrorStrategy 代理错误时的处理策略
	ProxyErrorStrategy PeStrategy `yaml:"proxy-error-strategy"`
	// ImagesQuality 图片质量
	ImagesQuality int `yaml:"images-quality"`
	// Strm strm 配置
	Strm *Strm `yaml:"strm"`
	// DownloadStrategy 下载接口响应策略
	DownloadStrategy DlStrategy `yaml:"download-strategy"`
	// LocalMediaRoot 本地媒体根路径
	LocalMediaRoot string `yaml:"local-media-root"`
}

func (e *Emby) Init() error {
	if strs.AnyEmpty(e.Host) {
		return errors.New("emby.host 配置不能为空")
	}
	if strs.AnyEmpty(e.MountPath) {
		return errors.New("emby.mount-path 配置不能为空")
	}
	if strs.AnyEmpty(string(e.ProxyErrorStrategy)) {
		// 失败默认回源
		e.ProxyErrorStrategy = PeStrategyOrigin
	}
	if strs.AnyEmpty(string(e.DownloadStrategy)) {
		// 默认响应直链
		e.DownloadStrategy = DlStrategyDirect
	}

	e.ProxyErrorStrategy = PeStrategy(strings.TrimSpace(string(e.ProxyErrorStrategy)))
	if _, ok := validPeStrategy[e.ProxyErrorStrategy]; !ok {
		return fmt.Errorf("emby.proxy-error-strategy 配置错误, 有效值: %v", maps.Keys(validPeStrategy))
	}

	if e.ImagesQuality == 0 {
		// 不允许配置零值
		e.ImagesQuality = 70
	}
	if e.ImagesQuality < 0 || e.ImagesQuality > 100 {
		return fmt.Errorf("emby.images-quality 配置错误: %d, 允许配置范围: [1, 100]", e.ImagesQuality)
	}

	if e.Strm == nil {
		e.Strm = new(Strm)
	}
	if err := e.Strm.Init(); err != nil {
		return fmt.Errorf("emby.strm 配置错误: %v", err)
	}

	e.DownloadStrategy = DlStrategy(strings.TrimSpace(string(e.DownloadStrategy)))
	if _, ok := validDlStrategy[e.DownloadStrategy]; !ok {
		return fmt.Errorf("emby.download-strategy 配置错误, 有效值: %v", maps.Keys(validDlStrategy))
	}

	// 如果没有配置, 生成一个随机前缀, 避免将网盘资源误识别为本地
	if e.LocalMediaRoot = strings.TrimSpace(e.LocalMediaRoot); e.LocalMediaRoot == "" {
		e.LocalMediaRoot = "/" + randoms.RandomHex(32)
	}

	return nil
}

// Strm strm 配置
type Strm struct {
	// PathMap 远程路径映射
	PathMap []string `yaml:"path-map"`
	// pathMap 配置初始化后转换为标准的 map 结构
	pathMap map[string]string
}

// Init 配置初始化
func (s *Strm) Init() error {
	s.pathMap = make(map[string]string)
	for _, path := range s.PathMap {
		splits := strings.Split(path, "=>")
		if len(splits) != 2 {
			return fmt.Errorf("映射配置不规范: %s, 请使用 => 进行分割", path)
		}
		from, to := strings.TrimSpace(splits[0]), strings.TrimSpace(splits[1])
		s.pathMap[from] = to
	}
	return nil
}

// MapPath 将传入路径按照预配置的映射关系从上到下按顺序进行映射,
// 至多成功映射一次
func (s *Strm) MapPath(path string) string {
	for from, to := range s.pathMap {
		if strings.Contains(path, from) {
			return strings.Replace(path, from, to, 1)
		}
	}
	return path
}
