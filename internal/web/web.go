package web

import (
	"go-emby2alist/internal/config"
	"go-emby2alist/internal/util/color"
	"go-emby2alist/internal/web/cache"
	"go-emby2alist/internal/web/webport"
	"log"

	"github.com/gin-gonic/gin"
)

// Listen 监听指定端口
func Listen() error {
	r := gin.Default()

	r.Use(referrerPolicySetter())
	if config.C.Cache.Enable {
		r.Use(cache.CacheableRouteMarker())
		r.Use(cache.RequestCacher())
	}

	initRoutes(r)
	initRulePatterns()

	errChanHTTP, errChanHTTPS := make(chan error, 1), make(chan error, 1)
	if !config.C.Ssl.Enable {
		go listenHTTP(r, errChanHTTP)
	} else if config.C.Ssl.SinglePort {
		go listenHTTPS(r, errChanHTTPS)
	} else {
		go listenHTTP(r, errChanHTTP)
		go listenHTTPS(r, errChanHTTPS)
	}

	select {
	case err := <-errChanHTTP:
		log.Fatal("http 服务异常: ", err)
	case err := <-errChanHTTPS:
		log.Fatal("https 服务异常: ", err)
	}
	return nil
}

// listenHTTP 在指定端口上监听 http 服务
//
// 出现错误时, 会写入 errChan 中
func listenHTTP(r *gin.Engine, errChan chan error) {
	log.Printf(color.ToBlue("在端口【%s】上启动 HTTP 服务"), webport.HTTP)
	err := r.Run("0.0.0.0:" + webport.HTTP)
	errChan <- err
	close(errChan)
}

// listenHTTPS 在指定端口上监听 https 服务
//
// 出现错误时, 会写入 errChan 中
func listenHTTPS(r *gin.Engine, errChan chan error) {
	log.Printf(color.ToBlue("在端口【%s】上启动 HTTPS 服务"), webport.HTTPS)
	ssl := config.C.Ssl
	err := r.RunTLS("0.0.0.0:"+webport.HTTPS, ssl.CrtPath(), ssl.KeyPath())
	errChan <- err
	close(errChan)
}
