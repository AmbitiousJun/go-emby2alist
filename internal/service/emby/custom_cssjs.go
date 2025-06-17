package emby

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/constant"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/https"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

// customJsList 首次访问时, 将所有自定义脚本预加载在内存中
var customJsList = []string{}

// customCssList 首次访问时, 将所有自定义样式预加载在内存中
var customCssList = []string{}

// loadAllCustomCssJs 加载所有自定义脚本
var loadAllCustomCssJs = sync.OnceFunc(func() {
	loadRemoteContent := func(originBytes []byte) ([]byte, error) {
		if len(originBytes) == 0 {
			return []byte{}, nil
		}

		str := strings.TrimSpace(string(originBytes))
		u, err := url.Parse(str)
		if err != nil {
			// 非远程地址
			return originBytes, nil
		}

		resp, err := https.Get(u.String()).Do()
		if err != nil {
			return nil, fmt.Errorf("远程加载失败: %s, err: %v", u.String(), err)
		}
		defer resp.Body.Close()
		if !https.IsSuccessCode(resp.StatusCode) {
			return nil, fmt.Errorf("远程错误响应: %s, err: %s", u.String(), resp.Status)
		}

		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("远程读取失败: %s, err: %v", u.String(), err)
		}

		return bytes, nil
	}

	loadFiles := func(fp, ext, successLogPrefix string) ([]string, error) {
		if err := os.MkdirAll(fp, os.ModePerm); err != nil {
			return nil, fmt.Errorf("目录初始化失败: %s, err: %v", fp, err)
		}

		files, err := os.ReadDir(fp)
		if err != nil {
			return nil, fmt.Errorf("读取目录失败: %s, err: %v, 无法注入自定义脚本", fp, err)
		}

		res := []string{}
		ch := make(chan string)
		chg := new(errgroup.Group)
		chg.Go(func() error {
			for content := range ch {
				res = append(res, content)
			}
			return nil
		})

		g := new(errgroup.Group)
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			if filepath.Ext(file.Name()) != ext {
				continue
			}

			g.Go(func() error {
				content, err := os.ReadFile(filepath.Join(fp, file.Name()))
				if err != nil {
					return fmt.Errorf("读取文件失败: %s, err: %v", file.Name(), err)
				}

				// 支持远程加载
				content, err = loadRemoteContent(content)
				if err != nil {
					return fmt.Errorf("远程加载失败: %s, err: %v", file.Name(), err)
				}

				ch <- string(content)
				log.Printf(colors.ToGreen("%s已加载: %s"), successLogPrefix, file.Name())
				return nil
			})

		}

		if err := g.Wait(); err != nil {
			close(ch)
			return nil, err
		}
		close(ch)
		chg.Wait()
		return res, nil
	}

	fp := filepath.Join(config.BasePath, constant.CustomJsDirName)
	jsList, err := loadFiles(fp, ".js", "自定义脚本")
	if err != nil {
		log.Printf(colors.ToRed("加载自定义脚本异常: %v"), err)
		return
	}
	customJsList = append(customJsList, jsList...)

	fp = filepath.Join(config.BasePath, constant.CustomCssDirName)
	cssList, err := loadFiles(fp, ".css", "自定义样式表")
	if err != nil {
		log.Printf(colors.ToRed("加载自定义样式表异常: %v"), err)
		return
	}
	customCssList = append(customCssList, cssList...)
})

// ProxyIndexHtml 代理 index.html 注入自定义脚本样式文件
func ProxyIndexHtml(c *gin.Context) {
	embyHost := config.C.Emby.Host
	resp, err := https.Request(c.Request.Method, embyHost+c.Request.URL.String()).
		Header(c.Request.Header).
		Body(c.Request.Body).
		Do()
	if checkErr(c, err) {
		return
	}
	defer resp.Body.Close()

	if !https.IsSuccessCode(resp.StatusCode) && checkErr(c, err) {
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if checkErr(c, err) {
		return
	}

	content := string(bodyBytes)
	bodyCloseTag := "</body>"
	customJsElm := fmt.Sprintf(`    <script src="%s"></script>`, constant.Route_CustomJs)
	content = strings.ReplaceAll(content, bodyCloseTag, customJsElm+"\n"+bodyCloseTag)

	customCssElm := fmt.Sprintf(`    <link rel="stylesheet" type="text/css" href="%s">`, constant.Route_CustomCss)
	content = strings.ReplaceAll(content, bodyCloseTag, customCssElm+"\n"+bodyCloseTag)

	c.Status(resp.StatusCode)
	resp.Header.Del("Content-Length")
	https.CloneHeader(c, resp.Header)
	c.Writer.Write([]byte(content))
	c.Writer.Flush()
}

// ProxyCustomJs 代理自定义脚本
func ProxyCustomJs(c *gin.Context) {
	loadAllCustomCssJs()

	contentBuilder := strings.Builder{}
	for _, script := range customJsList {
		contentBuilder.WriteString(fmt.Sprintf("(function(){ %s })();\n", script))
	}
	contentBytes := []byte(contentBuilder.String())

	c.Status(http.StatusOK)
	c.Header("Content-Type", "application/javascript")
	c.Header("Content-Length", fmt.Sprintf("%d", len(contentBytes)))
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Writer.Write(contentBytes)
	c.Writer.Flush()
}

// ProxyCustomCss 代理自定义样式表
func ProxyCustomCss(c *gin.Context) {
	loadAllCustomCssJs()

	contentBuilder := strings.Builder{}
	for _, style := range customCssList {
		contentBuilder.WriteString(fmt.Sprintf("%s\n\n\n", style))
	}
	contentBytes := []byte(contentBuilder.String())

	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/css")
	c.Header("Content-Length", fmt.Sprintf("%d", len(contentBytes)))
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Writer.Write(contentBytes)
	c.Writer.Flush()
}
