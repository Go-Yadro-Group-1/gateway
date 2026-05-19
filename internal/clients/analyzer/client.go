package analyzer

import (
	"fmt"
	"time"

	analyzerv1 "github.com/Go-Yadro-Group-1/gateway/gen/grpc/analyzer/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const (
	keepaliveTime    = 30 * time.Second
	keepaliveTimeout = 10 * time.Second
)

type Client struct {
	client  analyzerv1.AnalyzerServiceClient
	connRef *grpc.ClientConn
}

func Dial(address string) (*Client, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                keepaliveTime,
			Timeout:             keepaliveTimeout,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("dial analyzer %q: %w", address, err)
	}

	//nolint:ireturn // generated client constructor returns an interface
	return &Client{
		client:  analyzerv1.NewAnalyzerServiceClient(conn),
		connRef: conn,
	}, nil
}

//nolint:ireturn // generated client is consumed as an interface
func (c *Client) Analyzer() analyzerv1.AnalyzerServiceClient {
	return c.client
}

func (c *Client) Close() error {
	err := c.connRef.Close()
	if err != nil {
		return fmt.Errorf("close analyzer conn: %w", err)
	}

	return nil
}
