package config

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

// durationMap 字符串配置映射成 time.Duration
var durationMap = map[string]time.Duration{
	"d": time.Hour * 24,
	"h": time.Hour,
	"m": time.Minute,
	"s": time.Second,
}

type Cache struct {
	Enable  bool          `yaml:"enable"`  // 是否启用缓存
	Expired string        `yaml:"expired"` // 缓存过期时间
	expired time.Duration // 配置初始化转换之后的标准时间对象
}

func (c *Cache) ExpiredDuration() time.Duration {
	return c.expired
}

func (c *Cache) Init() error {
	if len(c.Expired) == 0 {
		// 缓存默认过期时间一天
		c.expired = time.Hour * 24
	} else {
		timeFlag := c.Expired[len(c.Expired)-1:]
		duration, ok := durationMap[timeFlag]
		if !ok {
			return fmt.Errorf("cache.expired 配置错误: %s, 支持的时间单位: s, m, h, d", timeFlag)
		}
		base, err := strconv.Atoi(c.Expired[:len(c.Expired)-1])
		if err != nil {
			return fmt.Errorf("cache.expired 配置错误: %v", err)
		}
		if base < 1 {
			return fmt.Errorf("cache.exipred 配置错误: %d, 值需大于 0", base)
		}
		c.expired = time.Duration(base) * duration
	}

	if c.Enable {
		log.Println("缓存中间件已启用, 过期时间: ", c.Expired)
	}

	return nil
}
