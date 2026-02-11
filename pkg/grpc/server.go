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
	processes map[int]*processContext
}

type processContext struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	done   chan struct{}
}

// NewWSLServer creates a new WSLServer instance
func NewWSLServer() *WSLServer {
	return &WSLServer{
		processes: make(map[int]*processContext),
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

	procCtx := &processContext{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		done:   make(chan struct{}),
	}

	// 等待进程结束
	go func() {
		cmd.Wait()
		close(procCtx.done)
	}()

	s.mu.Lock()
	s.processes[cmd.Process.Pid] = procCtx
	s.mu.Unlock()

	return &pb.StartResponse{Pid: int32(cmd.Process.Pid)}, nil
}

func (s *WSLServer) Stop(ctx context.Context, req *pb.StopRequest) (*pb.StopResponse, error) {
	s.mu.Lock()
	procCtx, ok := s.processes[int(req.Pid)]
	s.mu.Unlock()

	if !ok {
		return &pb.StopResponse{ExitCode: 0}, nil
	}

	// 关闭 stdin
	procCtx.stdin.Close()

	// 等待进程结束
	select {
	case <-procCtx.done:
	case <-ctx.Done():
		// 超时，强制 kill
		procCtx.cmd.Process.Kill()
		<-procCtx.done
	}

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
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// TODO: 发送输入到进程 (需要 PID)
		_ = req
	}
}

func (s *WSLServer) Stdout(req *pb.Empty, stream pb.DevPodWSLService_StdoutServer) error {
	_ = req
	return nil
}

func (s *WSLServer) Stderr(req *pb.Empty, stream pb.DevPodWSLService_StderrServer) error {
	_ = req
	return nil
}

func (s *WSLServer) Status(ctx context.Context, req *pb.Empty) (*pb.AgentStatus, error) {
	return &pb.AgentStatus{Running: true}, nil
}

func (s *WSLServer) Upload(stream pb.DevPodWSLService_UploadServer) error {
	return nil
}
