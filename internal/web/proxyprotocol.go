package web

import (
	"bufio"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/gin-gonic/gin"
	"github.com/pires/go-proxyproto"
)

// initProxyProtocolLn 初始化一个监听指定端口的兼容 proxy protocol 的 net.Listener 实例
func initProxyProtocolLn(port string) (net.Listener, error) {
	ln, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		return nil, err
	}
	return NewHybridListener(ln), nil
}

// proxyProtocolRealIPSetter 将 gin 上下文中的 RealIP 设置为 PROXY 协议中传递的 IP 地址的中间件
func proxyProtocolRealIPSetter() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf(colors.ToYellow("RemoteAddr: %s"), c.Request.RemoteAddr)
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err == nil {
			c.Request.Header.Set("X-Real-IP", host)
			c.Request.Header.Set("X-Forwarded-For", host)
		}
	}
}

// HybridListener wraps a net.Listener to support both PROXY protocol and plain TCP
type HybridListener struct {
	net.Listener
}

func NewHybridListener(ln net.Listener) net.Listener {
	return &HybridListener{Listener: ln}
}

func (hl *HybridListener) Accept() (net.Conn, error) {
	conn, err := hl.Listener.Accept()
	if err != nil {
		return nil, err
	}

	// peek the first few bytes
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	reader := bufio.NewReader(conn)
	peek, err := reader.Peek(32)
	// clear deadline
	conn.SetReadDeadline(time.Time{})

	if err != nil && err != io.EOF {
		conn.Close()
		return nil, err
	}

	str := string(peek)
	log.Printf(colors.ToYellow("TCP connection prefix: [%s]"), str)
	if strings.HasPrefix(str, "PROXY ") {
		// Wrap with proxyproto if starts with PROXY
		pxyConn := proxyproto.NewConn(&connWrapper{Conn: conn, reader: reader})
		log.Printf(colors.ToYellow("TCP connection RemoteAddr: [%s]"), pxyConn.RemoteAddr())
		log.Printf(colors.ToYellow("TCP connection LocalAddr: [%s]"), pxyConn.LocalAddr())
		return pxyConn, nil
	}

	// Otherwise, return plain connection
	return &connWrapper{Conn: conn, reader: reader}, nil
}

// connWrapper allows reusing the peeked data
type connWrapper struct {
	net.Conn
	reader *bufio.Reader
}

func (c *connWrapper) Read(b []byte) (int, error) {
	return c.reader.Read(b)
}
