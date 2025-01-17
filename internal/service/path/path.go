package path

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/AmbitiousJun/go-emby2alist/internal/config"
	"github.com/AmbitiousJun/go-emby2alist/internal/service/alist"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/colors"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/jsons"
	"github.com/AmbitiousJun/go-emby2alist/internal/util/urls"
)

// AlistPathRes 路径转换结果
type AlistPathRes struct {

	// Success 转换是否成功
	Success bool

	// Path 转换后的路径
	Path string

	// Range 遍历所有 Alist 根路径生成的子路径
	Range func() ([]string, error)
}

// Emby2Alist Emby 资源路径转 Alist 资源路径
func Emby2Alist(embyPath string) AlistPathRes {
	pathRoutes := strings.Builder{}
	pathRoutes.WriteString("[")
	pathRoutes.WriteString("\n1 => " + embyPath)

	embyPath = urls.TransferSlash(embyPath)
	pathRoutes.WriteString("\n2 => " + embyPath)

	embyMount := config.C.Emby.MountPath
	alistFilePath := strings.TrimPrefix(embyPath, embyMount)
	pathRoutes.WriteString("\n3 => " + alistFilePath)

	if mapPath, ok := config.C.Path.MapEmby2Alist(alistFilePath); ok {
		alistFilePath = mapPath
		pathRoutes.WriteString("\n4 => " + alistFilePath)
	}
	pathRoutes.WriteString("\n]")
	log.Printf(colors.ToGray("embyPath 转换路径: %s"), pathRoutes.String())

	rangeFunc := func() ([]string, error) {
		filePath, err := SplitFromSecondSlash(alistFilePath)
		if err != nil {
			return nil, fmt.Errorf("alistFilePath 解析异常: %s, error: %v", alistFilePath, err)
		}

		res := alist.FetchFsList("/", nil)
		if res.Code != http.StatusOK {
			return nil, fmt.Errorf("请求 alist fs list 接口异常: %s", res.Msg)
		}

		paths := make([]string, 0)
		content, ok := res.Data.Attr("content").Done()
		if !ok || content.Type() != jsons.JsonTypeArr {
			return nil, fmt.Errorf("alist fs list 接口响应异常, 原始响应: %v", jsons.NewByObj(res))
		}

		content.RangeArr(func(_ int, value *jsons.Item) error {
			if value.Attr("is_dir").Val() == false {
				return nil
			}
			newPath := fmt.Sprintf("/%s%s", value.Attr("name").Val(), filePath)
			paths = append(paths, newPath)
			return nil
		})

		return paths, nil
	}

	return AlistPathRes{
		Success: true,
		Path:    alistFilePath,
		Range:   rangeFunc,
	}
}

// SplitFromSecondSlash 找到给定字符串 str 中第二个 '/' 字符的位置
// 并以该位置为首字符切割剩余的子串返回
func SplitFromSecondSlash(str string) (string, error) {
	str = urls.TransferSlash(str)
	firstIdx := strings.Index(str, "/")
	if firstIdx == -1 {
		return "", fmt.Errorf("字符串不包含 /: %s", str)
	}

	secondIdx := strings.Index(str[firstIdx+1:], "/")
	if secondIdx == -1 {
		return "", fmt.Errorf("字符串只有单个 /: %s", str)
	}

	return str[secondIdx+firstIdx+1:], nil
}
