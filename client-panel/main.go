package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/frp-cloud-panel/client-panel/internal/api"
	"github.com/frp-cloud-panel/client-panel/internal/config"
	"github.com/frp-cloud-panel/client-panel/internal/frpc"
	"github.com/frp-cloud-panel/client-panel/internal/server"
)

func main() {
	// 加载配置
	cfgPath := "config.json"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}

	var cfg *config.Config
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		// 配置文件不存在，使用默认配置
		cfg = config.DefaultConfig()
		log.Printf("Using default config")
	} else {
		var err error
		cfg, err = config.LoadConfig(cfgPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	}

	// 创建 API 客户端（连接 Server Panel）
	apiClient := api.NewClient(cfg)

	// 创建 FRPC 管理器
	frpcManager := frpc.NewManager(cfg)

	// 创建本地 API 服务器
	localServer := server.NewServer(cfg, apiClient, frpcManager)

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down client panel...")
		if err := localServer.Shutdown(); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	// 启动本地服务器
	addr := fmt.Sprintf(":%d", cfg.LocalPort)
	log.Printf("Client Panel starting on %s", addr)
	log.Printf("Server Panel: %s", cfg.Server.Address)
	log.Printf("FRPC Path: %s", cfg.FRPC.Path)

	if err := localServer.Start(addr); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
