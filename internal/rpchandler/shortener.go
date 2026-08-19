package rpchandler

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/qutaq/short_url/internal/auth"
	"github.com/qutaq/short_url/internal/handler"
	"github.com/qutaq/short_url/internal/service"
	pb "github.com/qutaq/short_url/pkg/shortenerpb"
)

// ShortenerServer реализует gRPC-сервис сокращения ссылок.
type ShortenerServer struct {
	pb.UnimplementedShortenerServiceServer
	facade *handler.ShortenerFacade
}

// NewShortenerServer создаёт gRPC-обработчик на основе общего фасада.
func NewShortenerServer(facade *handler.ShortenerFacade) *ShortenerServer {
	return &ShortenerServer{facade: facade}
}

// ShortenURL сокращает переданный URL.
func (s *ShortenerServer) ShortenURL(ctx context.Context, req *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {
	result, err := s.facade.ShortenURL(ctx, req.GetUrl())
	if err != nil {
		return nil, grpcStatusFromError(err)
	}
	return &pb.URLShortenResponse{
		Result:   result.ShortURL,
		Conflict: result.Conflict,
	}, nil
}

// ExpandURL возвращает исходный URL по идентификатору короткой ссылки.
func (s *ShortenerServer) ExpandURL(ctx context.Context, req *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	originalURL, err := s.facade.ExpandURL(ctx, req.GetId())
	if err != nil {
		return nil, grpcStatusFromError(err)
	}
	return &pb.URLExpandResponse{Result: originalURL}, nil
}

// ListUserURLs возвращает URL аутентифицированного пользователя.
func (s *ShortenerServer) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*pb.UserURLsResponse, error) {
	urls, err := s.facade.ListUserURLs(ctx)
	if err != nil {
		return nil, grpcStatusFromError(err)
	}

	resp := &pb.UserURLsResponse{}
	if len(urls) == 0 {
		return resp, nil
	}

	resp.Url = make([]*pb.URLData, len(urls))
	for i, u := range urls {
		resp.Url[i] = &pb.URLData{
			ShortUrl:    u.ShortURL,
			OriginalUrl: u.OriginalURL,
		}
	}
	return resp, nil
}

func grpcStatusFromError(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, service.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, service.ErrDeleted):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, auth.ErrNoUserID):
		return status.Error(codes.Unauthenticated, "unauthorized")
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
