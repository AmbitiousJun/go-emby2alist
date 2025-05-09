package https_test

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"testing"
)

func TestRelativeRedirect(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://example.com/user/addr/index.html", nil)
	if err != nil {
		log.Fatalf("创建请求失败: %v", err)
	}
	loc := "new_addr/a/test.mp4"

	dirPath := path.Dir(req.URL.Path)
	loc = fmt.Sprintf("%s://%s%s/%s", req.URL.Scheme, req.URL.Host, dirPath, loc)
	log.Println(loc)
}
