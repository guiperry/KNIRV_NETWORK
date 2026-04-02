package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/knirv/hasher/internal/proto/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client v1.HasherServiceClient
}

type ExportStream struct {
	stream v1.HasherService_ExportSecurityDataClient
}

func NewClient(socketPath string) (*Client, error) {
	conn, err := grpc.NewClient(
		fmt.Sprintf("unix:%s", socketPath),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	return &Client{
		conn:   conn,
		client: v1.NewHasherServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) ExportSecurityData(ctx context.Context, req *v1.ExportRequest) (*ExportStream, error) {
	stream, err := c.client.ExportSecurityData(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("export: %w", err)
	}
	return &ExportStream{stream: stream}, nil
}

func (s *ExportStream) Recv() (*v1.EncryptedChunk, error) {
	return s.stream.Recv()
}

func (c *Client) TriggerTraining(ctx context.Context, req *v1.TrainingRequest) (*v1.TrainingResponse, error) {
	return c.client.TriggerTraining(ctx, req)
}

func (c *Client) GetTrainingStatus(ctx context.Context, req *v1.TrainingStatusRequest) (*v1.TrainingStatusResponse, error) {
	return c.client.GetTrainingStatus(ctx, req)
}

func (c *Client) GetUserRules(ctx context.Context, req *v1.RulesRequest) (*v1.RulesResponse, error) {
	return c.client.GetUserRules(ctx, req)
}

func (c *Client) ValidateAction(ctx context.Context, req *v1.ActionRequest) (*v1.ActionResponse, error) {
	return c.client.ValidateAction(ctx, req)
}

func (c *Client) StreamActivity(ctx context.Context, req *v1.StreamActivityRequest) (<-chan *v1.ActivityEvent, error) {
	stream, err := c.client.StreamActivity(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("stream activity: %w", err)
	}

	ch := make(chan *v1.ActivityEvent, 100)
	go func() {
		defer close(ch)
		for {
			event, err := stream.Recv()
			if err != nil {
				return
			}
			select {
			case ch <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

func (c *Client) IsAvailable() bool {
	conn, err := net.DialTimeout("unix", c.conn.Target()[5:], 100*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
