package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

type httpServer struct {
	server *http.Server
}

func (h *httpServer) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", "0.0.0.0:8080")
	if err != nil {
		return fmt.Errorf("failed to listen on the port 8080: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/computeMetadata/v1/instance/service-accounts/default/identity", http.HandlerFunc(handler))

	h.server = &http.Server{
		Handler: mux,
	}

	if err := h.server.Serve(lis); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("the http server has aborted: %w", err)
	}

	return nil
}

func (h *httpServer) Stop(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

func handler(w http.ResponseWriter, r *http.Request) {
	audience := r.URL.Query().Get("audience")
	now := time.Now().Unix()

	fmt.Fprintf(w, "identity-token-for-%s-%d", audience, now)
}
