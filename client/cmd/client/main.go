package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/frp-cloud-panel/client/internal/api"
	"github.com/frp-cloud-panel/client/internal/config"
	"github.com/frp-cloud-panel/client/internal/frpc"
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

	// 创建 API 客户端
	apiClient := api.NewClient(cfg)

	// 检查是否已注册
	if !cfg.Device.IsRegistered() {
		// 需要注册
		if err := registerDevice(cfg, apiClient); err != nil {
			log.Fatalf("Failed to register device: %v", err)
		}

		// 保存配置
		if err := config.SaveConfig(cfg, cfgPath); err != nil {
			log.Printf("Failed to save config: %v", err)
		}
	}

	// 创建 FRPC 管理器
	frpcManager := frpc.NewManager(cfg)

	// 启动心跳
	go startHeartbeat(cfg, apiClient, frpcManager)

	// 启动状态监控
	go startStatusMonitor(frpcManager)

	// 启动配置同步
	go startConfigSync(cfg, apiClient, frpcManager)

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Client Panel started")
	log.Printf("Device: %s (ID: %s)", cfg.Device.DeviceName, cfg.Device.ClientID)
	log.Println("Press Ctrl+C to exit")

	<-quit

	log.Println("Shutting down...")
	if err := frpcManager.Stop(); err != nil {
		log.Printf("Failed to stop FRPC: %v", err)
	}
	log.Println("Client Panel stopped")
}

// registerDevice 注册设备
func registerDevice(cfg *config.Config, apiClient *api.Client) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Device Registration ===")
	fmt.Println("Please enter your credentials to register this device.")
	fmt.Println()

	// 获取服务端地址
	fmt.Printf("Server address [%s]: ", cfg.Server.Address)
	serverAddr, _ := reader.ReadString('\n')
	serverAddr = strings.TrimSpace(serverAddr)
	if serverAddr != "" {
		cfg.Server.Address = serverAddr
	}

	// 获取用户名
	fmt.Print("Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	// 获取密码
	fmt.Print("Password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	// 获取设备名称
	hostname, _ := os.Hostname()
	fmt.Printf("Device name [%s]: ", hostname)
	deviceName, _ := reader.ReadString('\n')
	deviceName = strings.TrimSpace(deviceName)
	if deviceName == "" {
		deviceName = hostname
	}

	// 注册
	fmt.Println("\nRegistering device...")
	resp, err := apiClient.Register(username, password, deviceName)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	// 保存设备信息
	cfg.Device.ClientID = resp.ClientID
	cfg.Device.DeviceToken = resp.DeviceToken
	cfg.Device.DeviceName = deviceName

	fmt.Println("Device registered successfully!")
	fmt.Printf("Client ID: %s\n", resp.ClientID)

	return nil
}

// startHeartbeat 启动心跳
func startHeartbeat(cfg *config.Config, apiClient *api.Client, frpcManager *frpc.Manager) {
	for {
		// 发送心跳
		resp, err := apiClient.Heartbeat()
		if err != nil {
			log.Printf("Heartbeat failed: %v", err)
		} else {
			// 检查是否需要同步配置
			if resp.NeedSync {
				log.Println("Config sync needed")
				syncConfig(cfg, apiClient, frpcManager)
			}
		}

		// 等待 30 秒
		time.Sleep(30 * time.Second)
	}
}

// startStatusMonitor 启动状态监控
func startStatusMonitor(frpcManager *frpc.Manager) {
	for status := range frpcManager.StatusCh() {
		if !status.Running {
			log.Printf("FRPC stopped: %s", status.Error)
			// TODO: 尝试重启
		}
	}
}

// startConfigSync 启动配置同步
func startConfigSync(cfg *config.Config, apiClient *api.Client, frpcManager *frpc.Manager) {
	// 初始同步
	syncConfig(cfg, apiClient, frpcManager)

	// 定期检查
	for {
		time.Sleep(5 * time.Minute)
		syncConfig(cfg, apiClient, frpcManager)
	}
}

// syncConfig 同步配置
func syncConfig(cfg *config.Config, apiClient *api.Client, frpcManager *frpc.Manager) {
	// 获取最新配置
	configResp, err := apiClient.GetConfig()
	if err != nil {
		log.Printf("Failed to get config: %v", err)
		return
	}

	// 生成配置文件
	if err := frpc.GenerateConfig(cfg, configResp.Mappings, configResp.ServerAddr); err != nil {
		log.Printf("Failed to generate config: %v", err)
		return
	}

	// 如果 FRPC 正在运行，重启它
	if frpcManager.IsRunning() {
		if err := frpcManager.Restart(); err != nil {
			log.Printf("Failed to restart FRPC: %v", err)
			apiClient.ApplyConfig(configResp.Version, "error", err.Error())
			return
		}
	} else {
		// 启动 FRPC
		if err := frpcManager.Start(); err != nil {
			log.Printf("Failed to start FRPC: %v", err)
			apiClient.ApplyConfig(configResp.Version, "error", err.Error())
			return
		}
	}

	// 上报成功
	if err := apiClient.ApplyConfig(configResp.Version, "success", ""); err != nil {
		log.Printf("Failed to report config applied: %v", err)
	}
}
