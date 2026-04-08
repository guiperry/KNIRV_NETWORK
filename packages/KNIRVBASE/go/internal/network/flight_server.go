package network

import (
	"context"
	"fmt"
	"strings"

	stor "github.com/knirvcorp/knirvbase/go/internal/storage"
	"github.com/knirvcorp/knirvbase/go/pkg/nrv"
)

type FlightServer struct {
	storage *stor.NRVStorage
}

func NewFlightServer(storage *stor.NRVStorage) *FlightServer {
	return &FlightServer{
		storage: storage,
	}
}

func (s *FlightServer) StreamBrackets(ctx context.Context, ticket string) (<-chan *nrv.Bracket, error) {
	parts := strings.SplitN(ticket, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("flight: invalid ticket format, expected <stream>.<collection>")
	}
	streamType, collection := parts[0], parts[1]
	goldOnly := streamType == "gold"

	return s.storage.StreamBrackets(ctx, collection, goldOnly)
}

func (s *FlightServer) DoGet(ticket string) (<-chan *nrv.Bracket, error) {
	return s.StreamBrackets(context.Background(), ticket)
}
