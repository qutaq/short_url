package main

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"

	"github.com/soheilhy/cmux"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/qutaq/short_url/internal/handler"
	"github.com/qutaq/short_url/internal/middleware"
	"github.com/qutaq/short_url/internal/rpchandler"
	pb "github.com/qutaq/short_url/pkg/shortenerpb"
)

func newGRPCServer(h *handler.ShortenerHandler) *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.GRPCAuthUnaryInterceptor,
			middleware.RequireAuthUnaryInterceptor,
		),
	)
	pb.RegisterShortenerServiceServer(srv, rpchandler.NewShortenerServer(h.Facade()))
	return srv
}

func serveMultiplexed(listener net.Listener, httpSrv *http.Server, grpcSrv *grpc.Server) error {
	mux := cmux.New(listener)
	grpcListener := mux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	httpListener := mux.Match(cmux.Any())

	errCh := make(chan error, 3)
	go func() { errCh <- grpcSrv.Serve(grpcListener) }()
	go func() { errCh <- httpSrv.Serve(httpListener) }()
	go func() { errCh <- mux.Serve() }()

	err := <-errCh
	if errors.Is(err, cmux.ErrListenerClosed) || errors.Is(err, cmux.ErrServerClosed) {
		return nil
	}
	return err
}

func newListener(addr string, enableHTTPS bool, logger *zap.Logger) (net.Listener, error) {
	if !enableHTTPS {
		return net.Listen("tcp", addr)
	}

	tlsConfig, err := newTLSConfig()
	if err != nil {
		return nil, err
	}

	logger.Info("HTTPS enabled")
	return tls.Listen("tcp", addr, tlsConfig)
}
