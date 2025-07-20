package httpserver

import (
	"context"
	"net/http"
	"os"

	"go.uber.org/zap"
)

type Server struct {
	httpServer *http.Server
}

func Start(addr string, metricsHandler http.Handler, log *zap.Logger) *Server {
	if addr == "" {
		addr = "127.0.0.1:9000"
	}
	if log == nil {
		log = zap.NewNop()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/metrics", metricsHandler)

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server failed", zap.Error(err), zap.String("addr", addr))
			os.Exit(1)
		}
	}()

	return &Server{httpServer: srv}
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Addr() string {
	if s.httpServer == nil {
		return ""
	}
	return s.httpServer.Addr
}
