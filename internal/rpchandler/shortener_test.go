package rpchandler

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/qutaq/short_url/internal/auth"
	"github.com/qutaq/short_url/internal/handler"
	"github.com/qutaq/short_url/internal/middleware"
	"github.com/qutaq/short_url/internal/repository"
	"github.com/qutaq/short_url/internal/service"
	pb "github.com/qutaq/short_url/pkg/shortenerpb"
)

const testBaseURL = "http://localhost:8080"

func startTestGRPCServer(t *testing.T) (pb.ShortenerServiceClient, func()) {
	t.Helper()

	repo := repository.NewMemoryRepository()
	svc := service.NewShortener(repo, testBaseURL)
	h := handler.NewShortenerHandler(svc, nil)

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middleware.GRPCAuthUnaryInterceptor,
			middleware.RequireAuthUnaryInterceptor,
		),
	)
	pb.RegisterShortenerServiceServer(grpcServer, NewShortenerServer(h.Facade()))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc dial: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = listener.Close()
		svc.Close()
	}

	return pb.NewShortenerServiceClient(conn), cleanup
}

func TestGRPCShortenAndExpandURL(t *testing.T) {
	client, cleanup := startTestGRPCServer(t)
	defer cleanup()

	ctx := context.Background()
	shortenResp, err := client.ShortenURL(ctx, &pb.URLShortenRequest{Url: "https://example.com"})
	if err != nil {
		t.Fatalf("ShortenURL: %v", err)
	}
	if shortenResp.GetResult() == "" {
		t.Fatal("ShortenURL returned empty result")
	}

	shortID := shortenResp.GetResult()[len(testBaseURL)+1:]
	expandResp, err := client.ExpandURL(ctx, &pb.URLExpandRequest{Id: shortID})
	if err != nil {
		t.Fatalf("ExpandURL: %v", err)
	}
	if expandResp.GetResult() != "https://example.com" {
		t.Fatalf("ExpandURL result = %q, want %q", expandResp.GetResult(), "https://example.com")
	}
}

func TestGRPCListUserURLsRequiresAuth(t *testing.T) {
	client, cleanup := startTestGRPCServer(t)
	defer cleanup()

	_, err := client.ListUserURLs(context.Background(), &emptypb.Empty{})
	if err == nil {
		t.Fatal("ListUserURLs without auth error = nil, want error")
	}
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("ListUserURLs status = %v, want %v", status.Code(err), codes.Unauthenticated)
	}
}

func TestGRPCListUserURLsReturnsSavedURLs(t *testing.T) {
	client, cleanup := startTestGRPCServer(t)
	defer cleanup()

	userID, err := auth.GenerateUserID()
	if err != nil {
		t.Fatalf("GenerateUserID: %v", err)
	}
	token := auth.SignUserID(userID)

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", token))
	if _, err := client.ShortenURL(ctx, &pb.URLShortenRequest{Url: "https://grpc-user.example.com"}); err != nil {
		t.Fatalf("ShortenURL: %v", err)
	}

	listResp, err := client.ListUserURLs(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("ListUserURLs: %v", err)
	}
	if len(listResp.GetUrl()) != 1 {
		t.Fatalf("urls len = %d, want 1", len(listResp.GetUrl()))
	}
	if listResp.GetUrl()[0].GetOriginalUrl() != "https://grpc-user.example.com" {
		t.Fatalf("original url = %q, want %q", listResp.GetUrl()[0].GetOriginalUrl(), "https://grpc-user.example.com")
	}
}
