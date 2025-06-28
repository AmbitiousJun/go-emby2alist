package colors

// 日志颜色输出常量
const (
	Blue   = "\x1b[38;2;090;156;248m"
	Green  = "\x1b[38;2;126;192;080m"
	Yellow = "\x1b[38;2;220;165;080m"
	Red    = "\x1b[38;2;228;116;112m"
	Purple = "\x1b[38;2;160;186;250m"
	Gray   = "\x1b[38;2;145;147;152m"

	reset = "\x1b[0m"
)

// Enabler
type Enabler interface {

	// EnableColor 标记是否启用颜色输出
	EnableColor() bool
}

var enabler Enabler

// SetEnabler 设置颜色输出控制器
func SetEnabler(e Enabler) { enabler = e }

// ToBlue 将字符串转成蓝色
func ToBlue(str string) string {
	return wrapColor(Blue, str)
}

// ToGreen 将字符串转成绿色
func ToGreen(str string) string {
	return wrapColor(Green, str)
}

// ToYellow 将字符串转成黄色
func ToYellow(str string) string {
	return wrapColor(Yellow, str)
}

// ToRed 将字符串转成红色
func ToRed(str string) string {
	return wrapColor(Red, str)
}

// ToPurple 将字符串转成紫色
func ToPurple(str string) string {
	return wrapColor(Purple, str)
}

// ToGray 将字符串转成灰色
func ToGray(str string) string {
	return wrapColor(Gray, str)
}

// wrapColor 将字符串 str 包裹上指定颜色的 ANSI 字符
//
// 如果用户关闭了颜色输出, 则直接返回原字符串
func wrapColor(color, str string) string {
	if enabler != nil && !enabler.EnableColor() {
		return str
	}
	return color + str + reset
}
