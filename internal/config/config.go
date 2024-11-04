package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// Emby emby 相关配置
	Emby *Emby `yaml:"emby"`
	// Alist alist 相关配置
	Alist *Alist `yaml:"alist"`
	// VideoPreview 网盘转码链接代理配置
	VideoPreview *VideoPreview `yaml:"video-preview"`
	// Path 路径相关配置
	Path *Path `yaml:"path"`
	// Cache 缓存相关配置
	Cache *Cache `yaml:"cache"`
	// Ssl ssl 相关配置
	Ssl *Ssl `yaml:"ssl"`
	// Log 日志相关配置
	Log *Log `yaml:"log"`
}

// C 全局唯一配置对象
var C *Config

// BasePath 配置文件所在的基础路径
var BasePath string

type Initializer interface {
	// Init 配置初始化
	Init() error
}

// ReadFromFile 从指定文件中读取配置
func ReadFromFile(path string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	if err = initBasePath(path); err != nil {
		return fmt.Errorf("初始化 BasePath 失败: %v", err)
	}

	C = new(Config)
	if err := yaml.Unmarshal(bytes, C); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	cVal := reflect.ValueOf(C).Elem()
	for i := 0; i < cVal.NumField(); i++ {
		field := cVal.Field(i)

		// 为配置项初始化零值
		if field.Kind() == reflect.Ptr && field.IsNil() {
			elmType := field.Type().Elem()
			field.Set(reflect.New(elmType))
		}

		// 配置项初始化
		if i, ok := field.Interface().(Initializer); ok {
			if err := i.Init(); err != nil {
				return fmt.Errorf("初始化配置文件失败: %v", err)
			}
		}
	}

	return nil
}

// initBasePath 初始化 BasePath
func initBasePath(path string) error {
	if filepath.IsAbs(path) {
		BasePath = filepath.Dir(path)
		return nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	BasePath = filepath.Dir(absPath)
	return nil
}
