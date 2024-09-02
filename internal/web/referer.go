package web

import (
	"regexp"

	"github.com/gin-gonic/gin"
)

const RefererControlHeaderKey = "Referrer-Policy"

// needRefererChecker 对于特定接口, 允许添加 referer 头
func needRefererChecker() gin.HandlerFunc {
	patterns := []*regexp.Regexp{
		// 查询系统信息接口
		regexp.MustCompile(`(?i)^/.*system/info`),
	}

	return func(c *gin.Context) {
		for _, pattern := range patterns {
			if pattern.MatchString(c.Request.RequestURI) {
				c.Writer.Header().Del(RefererControlHeaderKey)
				break
			}
		}
	}
}

// referrerPolicySetter 设置代理的 Referrer 策略
func referrerPolicySetter() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header(RefererControlHeaderKey, "no-referrer")
	}
}
