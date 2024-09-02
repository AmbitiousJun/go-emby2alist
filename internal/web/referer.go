package web

import (
	"github.com/gin-gonic/gin"
)

const RefererControlHeaderKey = "Referrer-Policy"

// referrerPolicySetter 设置代理的 Referrer 策略
func referrerPolicySetter() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header(RefererControlHeaderKey, "no-referrer")
	}
}
