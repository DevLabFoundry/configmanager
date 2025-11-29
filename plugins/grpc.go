package plugins

import (
	"context"

	"github.com/DevLabFoundry/configmanager/v3/plugins/proto"
)

// GRPCClient is an implementation of KV that talks over RPC.
type GRPCClient struct{ client proto.TokenStoreClient }

func (m *GRPCClient) Get(key string, metadata []byte) (string, error) {
	resp, err := m.client.Value(context.Background(), &proto.TokenValueRequest{
		Token:    key,
		Metadata: metadata,
	})
	if err != nil {
		return "", err
	}

	return resp.Value, nil
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Impl TokenStore
}

func (m *GRPCServer) Get(
	ctx context.Context,
	req *proto.TokenValueRequest) (*proto.TokenValueResponse, error) {
	v, err := m.Impl.Get(req.)
	return &proto.GetResponse{Value: v}, err
}
