// pkg/version/version.go
package version

import "fmt"

// 这些变量将在编译时通过 ldflags 的 -X 选项进行设置。
// 它们必须是包级别的公共变量（首字母大写）才能被 -X 访问。
var (
	// GitCommit 保存编译时代码库的 Git 短哈希值。
	GitCommit string = "unknown" // 如果没有注入，则为默认值

	// BuildTime 保存编译时的时间戳 (UTC)。
	BuildTime string = "undefined" // 如果没有注入，则为默认值

	// AppVersion 保存应用程序的版本号。
	AppVersion string = "0.0.0-dev" // 如果没有注入，则为默认值
)

// GetVersionInfo 返回格式化的版本信息字符串。
func GetVersionInfo() string {
	return fmt.Sprintf("Version: %s, GitCommit: %s, BuildTime: %s", AppVersion, GitCommit, BuildTime)
}

// PrintVersionInfo 打印版本信息到标准输出。
func PrintVersionInfo() {
	fmt.Println(GetVersionInfo())
}
