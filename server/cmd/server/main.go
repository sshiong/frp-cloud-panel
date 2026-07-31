package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/frp-cloud-panel/server/internal/config"
	"github.com/frp-cloud-panel/server/internal/database"
	"github.com/frp-cloud-panel/server/internal/api"
)

func main() {
	// 加载配置
	cfgPath := "config.json"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	var cfg *config.Config
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		// 配置文件不存在，使用默认配置并保存
		cfg = config.DefaultConfig()
		if err := config.SaveConfig(cfg, cfgPath); err != nil {
			log.Fatalf("Failed to save default config: %v", err)
		}
		log.Printf("Created default config file: %s", cfgPath)
	} else {
		var err error
		cfg, err = config.LoadConfig(cfgPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	}

	// 初始化数据库
	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalf("Failed to init database: %v", err)
	}
	defer database.Close()

	// 创建并启动 API 服务器
	server := api.NewServer(cfg)

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		if err := server.Shutdown(); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	// 启动服务器
	addr := cfg.Server.GetAddress()
	log.Printf("Server starting on %s", addr)
	if err := server.Start(addr); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
