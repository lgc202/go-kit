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
	Server   *ServerConfig  `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

var cfg *config.Config[AppConfig]

func main() {
	var err error
	cfg, err = config.Load("./config.yaml",
		config.WithDefaults[AppConfig](map[string]any{
			"server.host": "0.0.0.0",
			"server.port": 8080,
			"log.level":   "info",
		}),
		config.WithEnv[AppConfig]("APP"),
	)
	if err != nil {
		log.Fatal(err)
	}

	initServer()
	initDatabase()
	initLogger()

	c := cfg.Get()
	fmt.Printf("服务器: %s:%d\n", c.Server.Host, c.Server.Port)
	fmt.Printf("数据库: %s@%s:%d/%s\n", c.Database.User, c.Database.Host, c.Database.Port, c.Database.DBName)
	fmt.Printf("日志级别: %s\n", c.Log.Level)

	fmt.Println("\n修改 config.yaml 将触发回调，Ctrl+C 退出")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

func initServer() {
	cfg.OnChange(func(old, new AppConfig) {
		if config.Changed(old.Server, new.Server) {
			log.Printf("[Server] 配置变更: %+v", new.Server)
		}
	})
}

func initDatabase() {
	cfg.OnChange(func(old, new AppConfig) {
		if config.Changed(old.Database, new.Database) {
			log.Printf("[Database] 配置变更，重建连接池...")
		}
	})
}

func initLogger() {
	cfg.OnChange(func(old, new AppConfig) {
		if config.Changed(old.Log, new.Log) {
			log.Printf("[Logger] 配置变更: %s -> %s", old.Log.Level, new.Log.Level)
		}
	})
}
