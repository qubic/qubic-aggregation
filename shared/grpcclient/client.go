package grpcclient

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewConnection(target string, options ...grpc.DialOption) (*grpc.ClientConn, error) {
	allOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	allOptions = append(allOptions, options...)

	connection, err := grpc.NewClient(target, allOptions...)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", target, err)
	}

	return connection, nil
}
