package config

import (
	"errors"
	"strings"
)

// Emby 相关配置
type Emby struct {
	// Emby 源服务器地址
	Host string `yaml:"host"`
	// rclone 或者 cd 的挂载目录
	MountPath string `yaml:"mount-path"`
	// emby api key, 在 emby 管理后台配置并获取
	ApiKey string `yaml:"api-key"`
}

func (e *Emby) Init() error {
	if e.Host = strings.TrimSpace(e.Host); e.Host == "" {
		return errors.New("emby.host 配置不能为空")
	}
	if e.MountPath = strings.TrimSpace(e.MountPath); e.MountPath == "" {
		return errors.New("emby.mount-path 配置不能为空")
	}
	if e.ApiKey = strings.TrimSpace(e.ApiKey); e.ApiKey == "" {
		return errors.New("emby.api-key 配置不能为空")
	}
	return nil
}
