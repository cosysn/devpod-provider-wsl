package grpc

import (
	"context"
	"net"
	"time"

	pb "github.com/cosysn/devpod-provider-wsl/pkg/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client gRPC 客户端
type Client struct {
	conn   *grpc.ClientConn
	client pb.DevPodWSLServiceClient
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

// Stdin 发送 stdin 数据（客户端流）
func (c *Client) Stdin(ctx context.Context) (pb.DevPodWSLService_StdinClient, error) {
	return c.client.Stdin(ctx)
}

// SendStdin 发送 stdin 数据到指定进程
func (c *Client) SendStdin(ctx context.Context, pid int32, data []byte) error {
	stream, err := c.Stdin(ctx)
	if err != nil {
		return err
	}
	if err := stream.Send(&pb.StdinRequest{Pid: pid, Content: data}); err != nil {
		return err
	}
	_, err = stream.CloseAndRecv()
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
