package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/qutaq/short_url/internal/auth"
	pb "github.com/qutaq/short_url/pkg/shortenerpb"
)

// GRPCAuthUnaryInterceptor извлекает токен авторизации из metadata и сохраняет
// идентификатор пользователя в контексте запроса.
func GRPCAuthUnaryInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(auth.AuthorizationMetadataKey); len(values) > 0 {
			if userID, err := auth.VerifyToken(values[0]); err == nil {
				ctx = auth.ContextWithUserID(ctx, userID)
			}
		}
	}
	return handler(ctx, req)
}

// RequireAuthUnaryInterceptor требует наличия идентификатора пользователя в контексте.
func RequireAuthUnaryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if info != nil && info.FullMethod == pb.ShortenerService_ListUserURLs_FullMethodName {
		if _, ok := auth.UserIDFromContext(ctx); !ok {
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}
	}
	return handler(ctx, req)
}
