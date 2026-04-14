package grpc

import (
	"context"
	"fmt"
	"time"

	hasherpb "github.com/knirvcorp/knirvserver/backend/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client hasherpb.HasherServiceClient
}

func NewClient(addr string) *Client {
	// For now, assume Unix socket if addr starts with '/', otherwise TCP
	var dialAddr string
	if addr[0] == '/' {
		dialAddr = fmt.Sprintf("unix:%s", addr)
	} else {
		dialAddr = addr
	}

	conn, err := grpc.NewClient(
		dialAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(fmt.Sprintf("dial hasher: %v", err))
	}

	return &Client{
		conn:   conn,
		client: hasherpb.NewHasherServiceClient(conn),
	}
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) ExportSecurityData(req *hasherpb.ExportRequest) <-chan *hasherpb.EncryptedChunk {
	stream, err := c.client.ExportSecurityData(context.Background(), req)
	if err != nil {
		panic(fmt.Sprintf("export stream: %v", err))
	}

	ch := make(chan *hasherpb.EncryptedChunk)
	go func() {
		defer close(ch)
		for {
			chunk, err := stream.Recv()
			if err != nil {
				return
			}
			ch <- chunk
		}
	}()

	return ch
}
