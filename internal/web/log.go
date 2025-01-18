package web

import (
	"fmt"
	"strconv"
	"time"

	"github.com/AmbitiousJun/go-emby2alist/internal/constant"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/https"
	"github.com/gin-gonic/gin"
)

func CustomLogger(port string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 处理请求
		c.Next()

		// 记录日志
		fmt.Printf("%s %s | %s | %s | %s | %s %s | %s %s\n",
			colors.ToYellow("[ge2a:"+constant.CurrentVersion+"]"),
			start.Format("2006-01-02 15:04:05"),
			colorStatusCode(c.Writer.Status()),
			time.Since(start),
			c.ClientIP(),
			colors.ToBlue(port),
			colors.ToBlue(c.GetString(MatchRouteKey)),
			colors.ToBlue(c.Request.Method),
			c.Request.RequestURI,
		)
	}
}

// colorStatusCode 将响应码打上颜色标记
func colorStatusCode(code int) string {
	str := strconv.Itoa(code)
	if https.IsSuccessCode(code) || https.IsRedirectCode(code) {
		return colors.ToGreen(str)
	}
	if https.IsErrorCode(code) {
		return colors.ToRed(str)
	}
	return colors.ToBlue(str)
}
