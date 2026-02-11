package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cosysn/devpod-provider-wsl/pkg/tunnel"
	"github.com/cosysn/devpod-provider-wsl/pkg/grpc"
	pb "github.com/cosysn/devpod-provider-wsl/pkg/grpc/proto"
	grpcLib "google.golang.org/grpc"
)

func main() {
	// 命令行参数
	socketPath := flag.String("socket", tunnel.DefaultSocketPath, "Unix socket path")
	flag.Parse()

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Printf("Agent starting...")
	log.Printf("Socket path: %s", *socketPath)

	// 创建 Unix socket server
	server := tunnel.NewUnixServer(*socketPath)
	if err := server.Listen(); err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	log.Printf("Listening on %s", *socketPath)

	// 创建 gRPC server
	grpcServer := grpcLib.NewServer()
	pb.RegisterDevPodWSLServiceServer(grpcServer, grpc.NewWSLServer())

	// 在 goroutine 中启动 gRPC server
	go func() {
		// 创建 tunnel listener (包装 Unix socket)
		listener := tunnel.NewUnixListener(server)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	log.Printf("Agent started")

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("Agent stopping...")
	grpcServer.GracefulStop()
	server.Close()
	log.Printf("Agent stopped")
}
