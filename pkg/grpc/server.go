package grpc

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sync"

	pb "github.com/cosysn/devpod-provider-wsl/pkg/grpc/proto"
)

// WSLServer implements the DevPodWSLServiceServer interface
type WSLServer struct {
	pb.UnimplementedDevPodWSLServiceServer
	processes map[int]*exec.Cmd
	mu        sync.Mutex
}

// NewWSLServer creates a new WSLServer instance
func NewWSLServer() *WSLServer {
	return &WSLServer{
		processes: make(map[int]*exec.Cmd),
	}
}

func (s *WSLServer) Start(ctx context.Context, req *pb.StartRequest) (*pb.StartResponse, error) {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", req.Command)
	cmd.Dir = req.Workdir

	// 设置环境变量
	for k, v := range req.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	// 捕获输出
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// 异步读取输出
	go func() {
		io.Copy(os.Stdout, stdout)
	}()
	go func() {
		io.Copy(os.Stderr, stderr)
	}()

	s.mu.Lock()
	s.processes[cmd.Process.Pid] = cmd
	s.mu.Unlock()

	return &pb.StartResponse{Pid: int32(cmd.Process.Pid)}, nil
}

func (s *WSLServer) Stop(ctx context.Context, req *pb.StopRequest) (*pb.StopResponse, error) {
	s.mu.Lock()
	cmd, ok := s.processes[int(req.Pid)]
	s.mu.Unlock()

	if !ok {
		return &pb.StopResponse{ExitCode: 0}, nil
	}

	cmd.Process.Kill()
	cmd.Wait()

	s.mu.Lock()
	delete(s.processes, int(req.Pid))
	s.mu.Unlock()

	return &pb.StopResponse{ExitCode: 0}, nil
}

func (s *WSLServer) Exec(stream pb.DevPodWSLService_ExecServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// 处理输入
		switch data := req.Data.(type) {
		case *pb.ExecRequest_Input:
			// TODO: 发送输入到进程
			_ = data
		case *pb.ExecRequest_Eof:
			// TODO: 关闭 stdin
			_ = data
		}

		// TODO: 返回输出
		resp := &pb.ExecResponse{
			Stdout: []byte{},
			Stderr: []byte{},
			Done:   false,
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}

func (s *WSLServer) Stdin(stream pb.DevPodWSLService_StdinServer) error {
	return nil
}

func (s *WSLServer) Stdout(req *pb.Empty, stream pb.DevPodWSLService_StdoutServer) error {
	return nil
}

func (s *WSLServer) Stderr(req *pb.Empty, stream pb.DevPodWSLService_StderrServer) error {
	return nil
}

func (s *WSLServer) Status(ctx context.Context, req *pb.Empty) (*pb.AgentStatus, error) {
	return &pb.AgentStatus{Running: true}, nil
}

func (s *WSLServer) Upload(stream pb.DevPodWSLService_UploadServer) error {
	return nil
}
