package middleware

import (
	"context"
	"log"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LogTechnicalErrorInterceptor struct{}

func (lte *LogTechnicalErrorInterceptor) GetInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	h, err := handler(ctx, req)
	if err != nil {
		statusError, _ := status.FromError(err)
		if statusError.Code() == codes.Internal || statusError.Code() == codes.Unknown {
			lastIndex := strings.LastIndex(info.FullMethod, "/")
			var method string
			if lastIndex > 1 && len(info.FullMethod) > lastIndex+1 {
				method = info.FullMethod[lastIndex+1:]
			} else {
				method = info.FullMethod
			}
			log.Printf("[ERROR] [%s] %s: %s. Request: %v", statusError.Code(), method, err.Error(), req)
		}
	}
	return h, err
}
