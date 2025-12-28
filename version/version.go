// Package version 提供版本信息管理功能。
// 支持通过 -ldflags 在构建时注入版本信息。
//
// 如需语义化版本比较，请使用官方库 golang.org/x/mod/semver
package version

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/gosuri/uitable"
)

var (
	// gitVersion 是语义化的版本号，格式为 vMAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]
	gitVersion = "v0.0.0-master+$Format:%h$"
	// buildDate 是 ISO8601 格式的构建时间, $(date -u +'%Y-%m-%dT%H:%M:%SZ') 命令的输出
	buildDate = "1970-01-01T00:00:00Z"
	// gitCommit 是 Git 的 SHA1 值，$(git rev-parse HEAD) 命令的输出
	gitCommit = "$Format:%H$"
	// gitTreeState 代表构建时 Git 仓库的状态，值为 clean 或 dirty
	gitTreeState = ""
	// buildUser 是执行构建的用户
	buildUser = ""
	// buildHost 是执行构建的主机名
	buildHost = ""
)

// Info 包含了版本信息
type Info struct {
	GitVersion   string `json:"gitVersion"`
	GitCommit    string `json:"gitCommit"`
	GitTreeState string `json:"gitTreeState,omitempty"`
	BuildDate    string `json:"buildDate"`
	BuildUser    string `json:"buildUser,omitempty"`
	BuildHost    string `json:"buildHost,omitempty"`
	GoVersion    string `json:"goVersion"`
	Compiler     string `json:"compiler"`
	Platform     string `json:"platform"`
}

// String 返回人性化的版本信息字符串
func (info Info) String() string {
	if info.GitTreeState == "dirty" {
		return info.GitVersion + "-dirty"
	}
	return info.GitVersion
}

// ShortString 返回简短的版本字符串，仅包含版本号
func (info Info) ShortString() string {
	return info.GitVersion
}

// ToJSON 以 JSON 格式返回版本信息
func (info Info) ToJSON() (string, error) {
	s, err := json.Marshal(info)
	if err != nil {
		return "", fmt.Errorf("failed to marshal version info: %w", err)
	}
	return string(s), nil
}

// ToJSONIndent 以格式化的 JSON 格式返回版本信息
func (info Info) ToJSONIndent() (string, error) {
	s, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal version info: %w", err)
	}
	return string(s), nil
}

// Text 将版本信息编码为 UTF-8 格式的文本，并返回
func (info Info) Text() string {
	table := uitable.New()
	table.RightAlign(0)
	table.MaxColWidth = 80
	table.Separator = " "
	table.AddRow("gitVersion:", info.GitVersion)
	table.AddRow("gitCommit:", info.GitCommit)
	if info.GitTreeState != "" {
		table.AddRow("gitTreeState:", info.GitTreeState)
	}
	table.AddRow("buildDate:", info.BuildDate)
	if info.BuildUser != "" {
		table.AddRow("buildUser:", info.BuildUser)
	}
	if info.BuildHost != "" {
		table.AddRow("buildHost:", info.BuildHost)
	}
	table.AddRow("goVersion:", info.GoVersion)
	table.AddRow("compiler:", info.Compiler)
	table.AddRow("platform:", info.Platform)

	return table.String()
}

// Get 返回详尽的代码库版本信息，用来标明二进制文件由哪个版本的代码构建
func Get() Info {
	return Info{
		GitVersion:   gitVersion,
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		BuildDate:    buildDate,
		BuildUser:    buildUser,
		BuildHost:    buildHost,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
