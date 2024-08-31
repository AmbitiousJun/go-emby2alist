package config

import (
	"fmt"
	"strings"
)

type Path struct {
	// Emby2Alist Emby 的路径前缀映射到 Alist 的路径前缀, 两个路径使用 : 符号隔开
	Emby2Alist []string `yaml:"emby2alist"`

	// emby2AlistMap 根据 Emby2Alist 转换成路径 map
	emby2AlistMap map[string]string
}

func (p *Path) Init() error {
	p.emby2AlistMap = make(map[string]string)
	for _, e2a := range p.Emby2Alist {
		arr := strings.Split(e2a, ":")
		if len(arr) != 2 {
			return fmt.Errorf("path.emby2alist 配置错误, %s 无法根据 ':' 进行分割", e2a)
		}
		p.emby2AlistMap[arr[0]] = arr[1]
	}
	return nil
}

// MapEmby2Alist 将 emby 路径映射成 alist 路径
func (p *Path) MapEmby2Alist(embyPath string) (string, bool) {
	for ep, ap := range p.emby2AlistMap {
		if strings.HasPrefix(embyPath, ep) {
			return strings.ReplaceAll(embyPath, ep, ap), true
		}
	}
	return "", false
}
