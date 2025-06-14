package httpserver

import (
    "context"
    "net"
    "net/http"
    "time"
)

type Logger interface {
    Info(msg string, fields ...any)
    Error(msg string, fields ...any)
}

type Server struct {
    addr string
    log  Logger
    srv  *http.Server
}

func New(addr string, log Logger) *Server {
    mux := http.NewServeMux()
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("ok"))
    })
    mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("metrics"))
    })

    return &Server{
        addr: addr,
        log:  log,
        srv:  &http.Server{Addr: addr, Handler: mux},
    }
}

func (s *Server) Start(ctx context.Context) error {
    ln, err := net.Listen("tcp", s.addr)
    if err != nil {
        return err
    }

    s.log.Info("http server listening", "addr", ln.Addr().String())
    errCh := make(chan error, 1)

    go func() {
        errCh <- s.srv.Serve(ln)
    }()

    select {
    case <-ctx.Done():
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        _ = s.srv.Shutdown(shutdownCtx)
        return nil
    case err := <-errCh:
        if err == http.ErrServerClosed {
            return nil
        }
        return err
    }
}
