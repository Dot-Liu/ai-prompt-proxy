package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/eolinker/ai-prompt-proxy/internal/admin"
	"github.com/eolinker/ai-prompt-proxy/internal/proxy"
	"github.com/eolinker/ai-prompt-proxy/internal/service"
)

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

	var wg sync.WaitGroup

	// 启动代理服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		proxyServer := proxy.NewServer(cfg)
		log.Printf("AI Prompt Proxy 启动在端口 %s", *proxyPort)
		if err := proxyServer.Start(*proxyPort); err != nil {
			log.Fatalf("启动代理服务器失败: %v", err)
		}
	}()

	// 启动管理API服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		adminServer, err := admin.NewAdminServerWithService(configService, *configDir)
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
		// 这里可以添加优雅关闭逻辑
		os.Exit(0)
	}()

	// 等待所有服务器
	wg.Wait()
}