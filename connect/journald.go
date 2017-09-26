package connect

import (
	"fmt"
	"io"
	"time"

	"github.com/vaitekunas/journal/logrpc"
	"google.golang.org/grpc"
)

// ToJournald connects to a log server backend
func ToJournald(host string, port int, service, instance, token string, timeout time.Duration) (io.WriteCloser, error) {

	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", host, port), grpc.WithPerRPCCredentials(&logrpc.TokenCred{
		IP:       getIP(),
		Service:  service,
		Instance: instance,
		Token:    token,
	}), grpc.WithInsecure()) // TODO: replace or make it an option

	if err != nil {
		return nil, fmt.Errorf("ConnectToLogServer: could not establish a gRPC connection :%s", err.Error())
	}

	return &remoteClient{
		timeout: timeout,
		close:   conn.Close,
		client:  logrpc.NewRemoteLoggerClient(conn),
	}, nil
}
