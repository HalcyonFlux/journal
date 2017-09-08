package logrpc

import "golang.org/x/net/context"

// TokenCred implements grpc.PerRPCCredentials and can be used for authentication
// via gRPC
type TokenCred struct {
	IP       string
	Service  string
	Instance string
	Token    string
}

// GetRequestMetadata returns request metadata
func (c *TokenCred) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"service":  c.Service,
		"instance": c.Instance,
		"token":    c.Token,
		"ip":       c.IP,
	}, nil
}

// RequireTransportSecurity returns transport security preferences
func (c *TokenCred) RequireTransportSecurity() bool {
	return false
}
