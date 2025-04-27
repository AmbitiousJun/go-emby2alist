package jsons

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// OkResp 返回 json 响应, 状态码 200
func OkResp(c *gin.Context, data *Item) {
	Resp(c, http.StatusOK, data)
}

// Resp 返回 json 响应
func Resp(c *gin.Context, code int, data *Item) {
	if data == nil {
		data = NewEmptyObj()
	}
	str := data.String()
	c.Header("Content-Length", strconv.Itoa(len(str)))
	c.String(code, str)
}
