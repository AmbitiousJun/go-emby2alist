package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AmbitiousJun/go-emby2openlist/internal/util/strs"
)

// SslDir ssl 证书存放目录名称
const SslDir = "ssl"

type Ssl struct {
	Enable     bool   `yaml:"enable"`      // 是否启用
	SinglePort bool   `yaml:"single-port"` // 是否使用单一端口
	Key        string `yaml:"key"`         // 服务器私钥名称
	Crt        string `yaml:"crt"`         // 证书名称
}

func (s *Ssl) Init() error {
	if !s.Enable {
		return nil
	}

	if err := initSslDir(); err != nil {
		return fmt.Errorf("初始化 ssl 目录失败: %v", err)
	}

	// 非空校验
	if strs.AnyEmpty(s.Crt) {
		return errors.New("ssl.crt 配置不能为空")
	}
	if strs.AnyEmpty(s.Key) {
		return errors.New("ssl.key 配置不能为空")
	}

	// 判断证书密钥是否存在
	if stat, err := os.Stat(s.CrtPath()); err != nil || stat.IsDir() {
		return fmt.Errorf("检测不到证书文件, err: %v", err)
	}
	if stat, err := os.Stat(s.KeyPath()); err != nil || stat.IsDir() {
		return fmt.Errorf("检测不到密钥文件, err: %v", err)
	}

	return nil
}

// CrtPath 获取 cert 证书的绝对路径
func (s *Ssl) CrtPath() string {
	return filepath.Join(BasePath, SslDir, s.Crt)
}

// KeyPath 获取密钥的绝对路径
func (s *Ssl) KeyPath() string {
	return filepath.Join(BasePath, SslDir, s.Key)
}

// initSslDir 初始化 ssl 目录
func initSslDir() error {
	absPath := filepath.Join(BasePath, SslDir)
	stat, err := os.Stat(absPath)

	// 目录已存在
	if err == nil && stat.IsDir() {
		return nil
	}

	return os.MkdirAll(absPath, os.ModeDir|os.ModePerm)
}
