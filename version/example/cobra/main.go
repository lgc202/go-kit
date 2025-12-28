// Package main 演示 version 包与 cobra 框架的集成
package main

import (
	"fmt"
	"os"

	"github.com/lgc202/go-kit/version"
	"github.com/spf13/cobra"
)

var (
	outputFormat string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "myapp",
		Short: "示例应用程序",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("应用程序正在运行...")
		},
	}

	// 添加 version 子命令
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			info := version.Get()
			switch outputFormat {
			case "json":
				jsonStr, err := info.ToJSONIndent()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					os.Exit(1)
				}
				fmt.Println(jsonStr)
			case "short":
				fmt.Println(info.ShortString())
			default:
				fmt.Println(info.Text())
			}
		},
	}

	versionCmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "输出格式 (text, json, short)")
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
