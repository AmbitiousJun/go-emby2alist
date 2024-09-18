package config

import (
	"errors"

	"github.com/AmbitiousJun/go-emby2alist/internal/util/strs"
)

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
	return nil
}
