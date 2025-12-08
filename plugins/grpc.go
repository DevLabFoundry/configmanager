package plugins

import (
	"context"

	"github.com/DevLabFoundry/configmanager/v3/plugins/proto"
)

// GRPCClient is the host process talking to the plugins
// i.e. the GRPCServer implementation of the TokenStore
type GRPCClient struct{ client proto.TokenStoreClient }

func (m *GRPCClient) Value(key string, metadata []byte) (string, error) {
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

func (m *GRPCServer) Value(
	ctx context.Context,
	req *proto.TokenValueRequest) (*proto.TokenValueResponse, error) {
	v, err := m.Impl.Value(req.Token, req.Metadata)
	return &proto.TokenValueResponse{Value: v}, err
}
