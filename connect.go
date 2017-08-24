package log

import (
	"encoding/json"
	"fmt"
	"github.com/vaitekunas/log/logrpc"
	"io"
	"time"

	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// remoteClient implements the io.Writer and logrpc.RemoteLoggerClient interfaces
// and is used to write log entries to a remote log server
type remoteClient struct {
	timeout time.Duration
	client  logrpc.RemoteLoggerClient
}

// Write sends the log via gRPC to the remote log server
func (r *remoteClient) Write(p []byte) (n int, err error) {

	// Call context with timeout
	ctx, _ := context.WithTimeout(context.Background(), r.timeout)

	// Unmarshal log entry
	newEntry := logEntry{}
	if err := json.Unmarshal(p, &newEntry); err != nil {
		return 0, fmt.Errorf("Write: could not unmarshal logEntry: %s", err.Error())
	}

	// Send log entry
	if _, err := r.client.RemoteLog(ctx, &logrpc.LogEntry{Entry: newEntry}); err != nil {
		return 0, fmt.Errorf("Write: failed to write log to remote backend: %s", err.Error())
	}

	select {
	case <-ctx.Done():
		if ctx.Err() != nil {
			return 0, fmt.Errorf("Write: writing to remote backend failed: %s", ctx.Err().Error())
		}
	}

	return len(p), nil
}

// ConnectToLogServer connects to a log server backend
func ConnectToLogServer(host string, port int, token string, timeout time.Duration) (io.Writer, error) {

	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", host, port), grpc.WithPerRPCCredentials(&logrpc.TokenCred{
		Token: token,
	}))

	if err != nil {
		return nil, fmt.Errorf("ConnectToLogServer: could not establish a gRPC connection :%s", err.Error())
	}

	return &remoteClient{
		timeout: timeout,
		client:  logrpc.NewRemoteLoggerClient(conn),
	}, nil
}

// ConnectToKafka connects to a kafka backend as a producer
func ConnectToKafka(host string, port int, topic string) (io.Writer, error) {

	return nil, nil
}
