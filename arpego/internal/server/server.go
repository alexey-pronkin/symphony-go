package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
	listener   net.Listener
}

func New(
	runtime Runtime,
	tasks TaskPlatform,
	delivery DeliveryInsights,
	observability Observability,
	workspaceScanner WorkspaceSecurityScanner,
	port int,
	dashboardDir string,
) *Server {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	return &Server{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           NewHandler(runtime, tasks, delivery, observability, workspaceScanner, dashboardDir),
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return err
	}
	s.listener = ln
	go func() {
		_ = s.httpServer.Serve(ln)
	}()
	return nil
}

func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.httpServer.Addr
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
