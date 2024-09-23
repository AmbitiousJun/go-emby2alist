package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AmbitiousJun/go-emby2alist/internal/util/strs"
)

type PeStrategy string

const (
	StrategyOrigin PeStrategy = "origin" // 回源
	StrategyReject PeStrategy = "reject" // 拒绝请求
)

// validPeStrategy 用于校验用户配置的策略是否合法
var validPeStrategy = map[PeStrategy]struct{}{
	StrategyOrigin: {}, StrategyReject: {},
}

// Emby 相关配置
type Emby struct {
	// Emby 源服务器地址
	Host string `yaml:"host"`
	// rclone 或者 cd 的挂载目录
	MountPath string `yaml:"mount-path"`
	// emby api key, 在 emby 管理后台配置并获取
	ApiKey string `yaml:"api-key"`
	// EpisodesUnplayPrior 在获取剧集列表时是否将未播资源优先展示
	EpisodesUnplayPrior bool `yaml:"episodes-unplay-prior"`
	// ResortRandomItems 是否对随机的 items 进行重排序
	ResortRandomItems bool `yaml:"resort-random-items"`
	// ProxyErrorStrategy 代理错误时的处理策略
	ProxyErrorStrategy PeStrategy `yaml:"proxy-error-strategy"`
	// ImagesQuality 图片质量
	ImagesQuality int `yaml:"images-quality"`
}

func (e *Emby) Init() error {
	if strs.AnyEmpty(e.Host) {
		return errors.New("emby.host 配置不能为空")
	}
	if strs.AnyEmpty(e.MountPath) {
		return errors.New("emby.mount-path 配置不能为空")
	}
	if strs.AnyEmpty(e.ApiKey) {
		return errors.New("emby.api-key 配置不能为空")
	}
	if strs.AnyEmpty(string(e.ProxyErrorStrategy)) {
		// 失败默认回源
		e.ProxyErrorStrategy = StrategyOrigin
	}

	e.ProxyErrorStrategy = PeStrategy(strings.TrimSpace(string(e.ProxyErrorStrategy)))
	if _, ok := validPeStrategy[e.ProxyErrorStrategy]; !ok {
		return errors.New("emby.proxy-error-strategy 配置错误")
	}

	if e.ImagesQuality == 0 {
		// 不允许配置零值
		e.ImagesQuality = 70
	}
	if e.ImagesQuality < 0 || e.ImagesQuality > 100 {
		return fmt.Errorf("emby.images-quality 配置错误: %d, 允许配置范围: [1, 100]", e.ImagesQuality)
	}
	return nil
}
