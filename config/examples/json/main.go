package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	config "github.com/lgc202/go-kit/config"
)

type AppConfig struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Log      LogConfig      `json:"log"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
}

type LogConfig struct {
	Level string `json:"level"`
}

var cfg *config.Config[AppConfig]

func main() {
	var err error
	cfg, err = config.Load[AppConfig]("./config.json")
	if err != nil {
		log.Fatal(err)
	}

	cfg.OnChange(func(old, new AppConfig) {
		if config.Changed(old.Server, new.Server) {
			log.Printf("[Server] 配置变更: %+v", new.Server)
		}
		if config.Changed(old.Database, new.Database) {
			log.Printf("[Database] 配置变更")
		}
		if config.Changed(old.Log, new.Log) {
			log.Printf("[Logger] 配置变更: %s -> %s", old.Log.Level, new.Log.Level)
		}
	})

	c := cfg.Get()
	fmt.Printf("服务器: %s:%d\n", c.Server.Host, c.Server.Port)
	fmt.Printf("数据库: %s@%s:%d/%s\n", c.Database.User, c.Database.Host, c.Database.Port, c.Database.DBName)
	fmt.Printf("日志级别: %s\n", c.Log.Level)

	fmt.Println("\n修改 config.json 将触发回调，Ctrl+C 退出")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
