package web

import (
	"log"
	"regexp"

	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/web/webport"

	"github.com/gin-gonic/gin"
)

// globalDftHandler 全局默认兜底的请求处理器
func globalDftHandler(c *gin.Context) {
	// 依次匹配路由规则, 找到其他的处理器
	for _, rule := range rules {
		reg := rule[0].(*regexp.Regexp)
		if reg.MatchString(c.Request.RequestURI) {
			servePort, _ := c.Get(webport.GinKey)
			log.Printf(colors.ToBlue("监听端口: %s, 匹配路由: %s"), servePort, reg.String())
			rule[1].(gin.HandlerFunc)(c)
			return
		}
	}
}

// compileRules 编译路由的正则表达式
func compileRules(rs [][2]interface{}) [][2]interface{} {
	newRs := make([][2]interface{}, 0)
	for _, rule := range rs {
		reg, err := regexp.Compile(rule[0].(string))
		if err != nil {
			log.Printf("路由正则编译失败, pattern: %v, error: %v", rule[0], err)
			continue
		}
		rule[0] = reg

		rawHandler, ok := rule[1].(func(*gin.Context))
		if !ok {
			log.Printf("错误的请求处理器, pattern: %v", rule[0])
			continue
		}
		var handler gin.HandlerFunc = rawHandler
		rule[1] = handler
		newRs = append(newRs, rule)
	}
	return newRs
}
