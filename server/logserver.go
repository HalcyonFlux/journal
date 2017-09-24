package server

import (
  "io"
  "github.com/vaitekunas/journal/logrpc"
  context "golang.org/x/net/context"
)

// LogServer is the main interface implemented by journal/server
type LogServer interface {

  // AddDestination adds a new destination/backend
  AddDestination(name string, writer io.Writer) error

  // Lists all destinations/backends
  ListDestinations() []string

  // RemoveDestination removes a destination/backend
  RemoveDestination(name string) error

 // AddToken creates a new token for the service/instance if it does not yet exist
 AddToken(service, instance string) (string, error)

 // AggregateServiceStatistics aggregates statistics
 AggregateServiceStatistics() (totalVolume int64, services []*AggregateStatistics, hourly [24][2]int64)

 // Authorize is a gRPC interceptor that authorizes incoming RPCs
 Authorize(ctx context.Context) error

 // GatherStatistics saves log-related statistics
 GatherStatistics(service, instance, key, ip string, logEntry *logrpc.LogEntry)

 // GetStatistics returns LogServer's statistics
 GetStatistics() map[string]*Statistic

 // GetTokens returns LogServer's authentication tokens
 GetTokens() map[string]string

 // KillSwitch returns the internal killswitch
 KillSwitch() chan bool

 // Logfiles returns statistics about available log files
 Logfiles() (map[string]string, error)

 // Quit stops the server and all goroutines
 Quit()

 // RemoteLog handles incoming remote logs
 RemoteLog(ctx context.Context, logEntry *logrpc.LogEntry) (*logrpc.Nothing, error)

 // RemoveToken removes an authentication token
 RemoveToken(service, instance string, lock bool) error

 // RemoveTokens removes all the authentication tokens of a service
 RemoveTokens(service string) error

}
