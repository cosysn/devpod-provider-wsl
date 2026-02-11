package grpc

import (
	"context"
	"net"
	"sync"
	"time"

	pb "github.com/cosysn/devpod-provider-wsl/pkg/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client gRPC 客户端
type Client struct {
	conn      *grpc.ClientConn
	client    pb.DevPodWSLServiceClient
	stdinLock sync.Mutex
	stdinStream pb.DevPodWSLService_StdinClient
}

// NewClient 创建 gRPC 客户端，连接到 Unix socket
func NewClient(socketPath string, timeout time.Duration) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		return net.DialTimeout("unix", socketPath, timeout)
	}

	conn, err := grpc.DialContext(
		ctx,
		"passthrough:///unix",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(dialer),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		client: pb.NewDevPodWSLServiceClient(conn),
	}, nil
}

// Start 启动命令
func (c *Client) Start(ctx context.Context, command, workdir string, env map[string]string) (*pb.StartResponse, error) {
	return c.client.Start(ctx, &pb.StartRequest{
		Command: command,
		Workdir: workdir,
		Env:     env,
	})
}

// Stop 停止进程
func (c *Client) Stop(ctx context.Context, pid int32) (*pb.StopResponse, error) {
	return c.client.Stop(ctx, &pb.StopRequest{Pid: pid})
}

// Exec 执行命令（双向流）
func (c *Client) Exec(ctx context.Context) (pb.DevPodWSLService_ExecClient, error) {
	return c.client.Exec(ctx)
}

// OpenStdin 打开 stdin 流
func (c *Client) OpenStdin(ctx context.Context) error {
	c.stdinLock.Lock()
	defer c.stdinLock.Unlock()

	stream, err := c.client.Stdin(ctx)
	if err != nil {
		return err
	}
	c.stdinStream = stream
	return nil
}

// SendStdin 发送 stdin 数据（需要先调用 OpenStdin）
func (c *Client) SendStdin(pid int32, data []byte) error {
	c.stdinLock.Lock()
	defer c.stdinLock.Unlock()

	if c.stdinStream == nil {
		return nil
	}
	return c.stdinStream.Send(&pb.StdinRequest{Pid: pid, Content: data})
}

// CloseStdin 关闭 stdin 流
func (c *Client) CloseStdin() error {
	c.stdinLock.Lock()
	defer c.stdinLock.Unlock()

	if c.stdinStream == nil {
		return nil
	}
	_, err := c.stdinStream.CloseAndRecv()
	c.stdinStream = nil
	return err
}

// Status 获取 agent 状态
func (c *Client) Status(ctx context.Context) (*pb.AgentStatus, error) {
	return c.client.Status(ctx, &pb.Empty{})
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.conn.Close()
}
