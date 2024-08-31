package web

import (
	"fmt"
	"go-emby2alist/internal/web/cache"

	"github.com/gin-gonic/gin"
)

// Listen 监听指定端口
func Listen(port int) error {
	r := gin.Default()

	r.Use(referrerPolicySetter())
	r.Use(cache.NopChecker())
	r.Use(cache.RequestCacher())

	initRoutes(r)

	if err := r.Run(fmt.Sprintf("0.0.0.0:%d", port)); err != nil {
		return fmt.Errorf("服务异常: %v", err)
	}
	return nil
}
