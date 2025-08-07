package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/eolinker/ai-prompt-proxy/internal/admin"
	"github.com/eolinker/ai-prompt-proxy/internal/logger"
	"github.com/eolinker/ai-prompt-proxy/internal/proxy"
	"github.com/eolinker/ai-prompt-proxy/internal/service"
)

// initDefaultLogger 初始化默认日志记录器
func initDefaultLogger() {
	// 创建默认的日志配置
	defaultConfig := logger.OutputConfig{
		Name:        "default",
		Driver:      "file",
		Description: "默认访问日志",
		Enabled:     true,
		Type:        logger.FormatterJSON,
		File:        "access.log",
		Dir:         "./logs",
		Period:      logger.PeriodHour,
		Expire:      3, // 保留30天
		Formatter: logger.FormatterConfig{
			Fields: map[string][]string{
				"default": {
					"$request_id", "$timestamp", "$method", "$path", "$user_agent",
					"$client_ip", "$api_key", "$user_id", "$request_size", "$request_body",
					"$model_id", "$target_model", "$proxy_url", "$proxy_scheme", "$proxy_host",
					"$upstream_body", "$status_code", "$response_size", "$response_time",
					"$response_body", "$error",
				},
			},
		},
	}

	// 添加默认日志记录器
	if err := logger.GlobalLoggerManager.AddLogger("default", defaultConfig); err != nil {
		log.Printf("初始化默认日志记录器失败: %v", err)
	} else {
		log.Println("默认日志记录器初始化成功")
	}
}

func main() {
	var (
		configDir = flag.String("config", "./configs", "配置文件目录")
		proxyPort = flag.String("proxy-port", "8080", "代理服务器端口")
		adminPort = flag.String("admin-port", "8081", "管理API端口")
	)
	flag.Parse()

	// 创建配置服务
	configService, err := service.NewConfigService(*configDir)
	if err != nil {
		log.Fatalf("创建配置服务失败: %v", err)
	}
	defer configService.Close()

	// 获取配置
	cfg := configService.GetConfig()

	// 创建认证服务（代理服务器需要用到）
	authService, err := service.NewAuthService(configService.GetDBManager())
	if err != nil {
		log.Fatalf("创建认证服务失败: %v", err)
	}

	// 初始化默认日志记录器
	initDefaultLogger()

	var wg sync.WaitGroup

	// 启动代理服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		proxyServer := proxy.NewServer(cfg, authService)
		log.Printf("AI Prompt Proxy 启动在端口 %s", *proxyPort)
		if err := proxyServer.Start(*proxyPort); err != nil {
			log.Fatalf("启动代理服务器失败: %v", err)
		}
	}()

	// 启动管理API服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		adminServer, err := admin.NewAdminServerWithService(configService, *configDir, *proxyPort, *adminPort)
		if err != nil {
			log.Fatalf("创建管理API服务器失败: %v", err)
		}
		log.Printf("管理API服务器启动在端口 %s", *adminPort)
		if err := adminServer.Start(*adminPort); err != nil {
			log.Fatalf("启动管理API服务器失败: %v", err)
		}
	}()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号或服务器退出
	go func() {
		<-sigChan
		log.Println("收到退出信号，正在关闭服务...")

		// 关闭日志记录器
		if err := logger.GlobalLoggerManager.Close(); err != nil {
			log.Printf("关闭日志记录器失败: %v", err)
		} else {
			log.Println("日志记录器已关闭")
		}

		os.Exit(0)
	}()

	// 等待所有服务器
	wg.Wait()
}
