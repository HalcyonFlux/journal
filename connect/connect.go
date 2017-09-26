package connect

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/vaitekunas/journal/logrpc"

	context "golang.org/x/net/context"
)

// remoteClient implements the io.Writer and logrpc.RemoteLoggerClient interfaces
// and is used to write log entries to a remote log server
type remoteClient struct {
	timeout time.Duration
	close   func() error
	client  logrpc.RemoteLoggerClient
}

// Write sends the log via gRPC to the remote log server
func (r *remoteClient) Write(p []byte) (n int, err error) {

	// Call context with timeout
	ctx, _ := context.WithTimeout(context.Background(), r.timeout)

	// Unmarshal log entry
	newEntry := map[int64]string{}
	if err := json.Unmarshal(p, &newEntry); err != nil {
		return 0, fmt.Errorf("Write: could not unmarshal logEntry: %s", err.Error())
	}

	// Send log entry
	if _, err := r.client.RemoteLog(ctx, &logrpc.LogEntry{Entry: newEntry}); err != nil {
		return 0, fmt.Errorf("Write: failed to write log to remote backend: %s", err.Error())
	}

	return len(p), nil
}

// Close closes the remote client connection
func (r *remoteClient) Close() error {
	if r.close != nil {
		return r.close()
	}
	return nil
}
