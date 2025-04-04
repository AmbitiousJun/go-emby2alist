package emby

import (
	"fmt"
	"io"
	"log"
	"net/http"
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

// customJsList 程序启动时, 将所有自定义脚本预加载在内存中
var customJsList = []string{}

// loadAllCustomJs 加载所有自定义脚本
var loadAllCustomJs = sync.OnceFunc(func() {
	fp := filepath.Join(config.BasePath, constant.CustomJsDirName)
	if err := os.MkdirAll(fp, os.ModePerm); err != nil {
		log.Printf(colors.ToRed("目录初始化失败: %s, err: %v"), fp, err)
		return
	}

	files, err := os.ReadDir(fp)
	if err != nil {
		log.Printf(colors.ToRed("读取目录失败: %s, err: %v, 无法注入自定义脚本"), fp, err)
		return
	}

	ch := make(chan string, len(files))
	go func() {
		for content := range ch {
			customJsList = append(customJsList, content)
		}
	}()

	g := new(errgroup.Group)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if filepath.Ext(file.Name()) != ".js" {
			continue
		}

		g.Go(func() error {
			content, err := os.ReadFile(filepath.Join(fp, file.Name()))
			if err != nil {
				return fmt.Errorf("读取文件失败: %s, err: %v", file.Name(), err)
			}
			ch <- string(content)
			log.Printf(colors.ToGreen("自定义脚本已加载: %s"), file.Name())
			return nil
		})

	}

	if err := g.Wait(); err != nil {
		log.Printf(colors.ToRed("读取脚本异常: %v"), err)
	}
	close(ch)

})

// ProxyIndexHtml 代理 index.html 注入自定义 js 脚本文件
func ProxyIndexHtml(c *gin.Context) {
	embyHost := config.C.Emby.Host
	resp, err := https.Request(c.Request.Method, embyHost+c.Request.URL.String(), c.Request.Header, c.Request.Body)
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
	customJsTag := fmt.Sprintf(`    <script src="%s"></script>`, constant.Route_CustomJs)
	content = strings.ReplaceAll(content, bodyCloseTag, customJsTag+"\n"+bodyCloseTag)

	c.Status(resp.StatusCode)
	resp.Header.Del("Content-Length")
	https.CloneHeader(c, resp.Header)
	c.Writer.Write([]byte(content))
	c.Writer.Flush()
}

// ProxyCustomJs 代理自定义脚本
func ProxyCustomJs(c *gin.Context) {
	loadAllCustomJs()

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
