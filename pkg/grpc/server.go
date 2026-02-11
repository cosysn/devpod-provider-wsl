package grpc

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
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

	// 创建 PTY
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	// 异步读取输出到 os.Stdout/os.Stderr
	go func() {
		io.Copy(os.Stdout, ptyFile)
		ptyFile.Close()
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

	// 启动 PTY shell
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Env = os.Environ()

	// 创建 PTY
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	defer ptyFile.Close()

	// 异步转发 stdin 到 PTY (过滤 Windows 换行符 CR)
	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				break
			}
			switch data := req.Data.(type) {
			case *pb.ExecRequest_Input:
				// 过滤掉 \r 字符 (Windows 换行符 CRLF -> LF)
				input := filterCR(data.Input)
				if len(input) > 0 {
					ptyFile.Write([]byte(input))
				}
			case *pb.ExecRequest_Eof:
				ptyFile.Close()
			}
		}
	}()

	// 读取 PTY 输出 (过滤 Windows 换行符 CR)
	buf := make([]byte, 4096)
	for {
		n, err := ptyFile.Read(buf)
		if n > 0 {
			// 过滤掉 \r 字符
			filtered := filterCRBytes(buf[:n])
			if len(filtered) > 0 {
				stream.Send(&pb.ExecResponse{Stdout: filtered})
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}

	cmd.Wait()
	stream.Send(&pb.ExecResponse{Done: true})

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

// filterCR 过滤掉 Windows 换行符中的 \r
func filterCR(input string) string {
	result := make([]byte, 0, len(input))
	for i := 0; i < len(input); i++ {
		if input[i] != '\r' {
			result = append(result, input[i])
		}
	}
	return string(result)
}

// filterCRBytes 过滤字节数组中的 \r
func filterCRBytes(input []byte) []byte {
	result := make([]byte, 0, len(input))
	for i := 0; i < len(input); i++ {
		if input[i] != '\r' {
			result = append(result, input[i])
		}
	}
	return result
}
