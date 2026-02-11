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
	mu        sync.Mutex
	processes map[int]*exec.Cmd
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

	// 创建 stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	// 创建 stdout/stderr pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, err
	}

	// 异步读取输出到 os.Stdout/os.Stderr
	go func() {
		io.Copy(os.Stdout, stdout)
		stdout.Close()
	}()
	go func() {
		io.Copy(os.Stderr, stderr)
		stderr.Close()
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
	// 解析命令
	req, err := stream.Recv()
	if err != nil {
		return err
	}

	// 获取命令
	var command string
	switch data := req.Data.(type) {
	case *pb.ExecRequest_Input:
		command = data.Input
	}

	// 启动 shell 执行命令
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Dir = ""
	cmd.Env = os.Environ()

	// 创建 pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return err
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return err
	}

	// 异步转发 stdin
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				break
			}
			switch data := req.Data.(type) {
			case *pb.ExecRequest_Input:
				stdin.Write([]byte(data.Input))
			case *pb.ExecRequest_Eof:
				stdin.Close()
			}
		}
	}()

	// 读取输出
	buf := make([]byte, 4096)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			stream.Send(&pb.ExecResponse{Stdout: buf[:n]})
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}

	cmd.Wait()
	stdout.Close()
	stderr.Close()

	return nil
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
