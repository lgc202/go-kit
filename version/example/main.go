// Package main 演示 version 包的使用方法
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lgc202/go-kit/version"
)

var (
	showVersion = flag.Bool("version", false, "显示版本信息")
	showJSON    = flag.Bool("json", false, "以 JSON 格式显示版本信息")
)

func main() {
	flag.Parse()

	if *showVersion {
		info := version.Get()
		if *showJSON {
			jsonStr, err := info.ToJSONIndent()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(jsonStr)
		} else {
			fmt.Println(info.Text())
		}
		os.Exit(0)
	}

	// 正常的应用逻辑
	fmt.Println("应用程序正在运行...")
	fmt.Printf("当前版本: %s\n", version.Get().String())
}
