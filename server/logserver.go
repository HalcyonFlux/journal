package server

import (
	"fmt"
	"github.com/vaitekunas/log"
	"github.com/vaitekunas/log/logrpc"
	"net"
	"os"
	"sync"
	"time"

	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	metadata "google.golang.org/grpc/metadata"
)

// killswitch is used to close all goroutines
type killswitch chan<- bool

// LogServer implements log.Logger and log.RemoteLoggerServer interfaces
type LogServer struct {
	logger *log.Logger    // Local logger
	server *grpc.Server   // gRPC server
	wg     sync.WaitGroup // Waitgroup for the unix and grpc listeners

	unixSockPath string       // Path to the unix socket file
	listenUnix   net.Listener // Unix-socket listener (unix)
	listenTCP    net.Listener // TCP listener (grpc)

	killswitches []killswitch

	tokens   map[string]map[string]string // Authorization tokens map[service]map[instance]token
	quitChan chan bool                    // Internal kill switch
}

// RemoteLog handles incoming remote logs
func (l *LogServer) RemoteLog(ctx context.Context, logEntry *logrpc.LogEntry) (*logrpc.Nothing, error) {
	if err := l.logger.RawEntry(logEntry.Entry); err != nil {
		return nil, fmt.Errorf("RemoteLog: could not process raw log: %s", err.Error())
	}
	return nil, nil
}

// Authorize is a gRPC interceptor that authorizes incoming RPCs
func (l *LogServer) Authorize(ctx context.Context) error {

	// Verify presence of metadata
	md, ok := metadata.FromContext(ctx)
	if !ok {
		return fmt.Errorf("Authorize: missing metadata")
	}

	// Verify that all required items are available
	for _, key := range []string{"service", "instance", "token"} {
		if slice, okKey := md[key]; !okKey || len(slice) != 1 {
			return fmt.Errorf("Authorize: missing %s", key)
		}
	}

	// Extract the real token
	service := md["service"][0]
	instance := md["instance"][0]
	token := md["token"][0]

	serviceTokens, ok := l.tokens[service]
	if !ok {
		return fmt.Errorf("Authorize: unknown service")
	}
	realToken, ok := serviceTokens[instance]
	if !ok {
		return fmt.Errorf("Authorize: unknown instance")
	}

	// Authorize
	if realToken != token {
		return fmt.Errorf("Authorize: bad token")
	}

	return nil
}

// KillSwitch returns the internal killswitch
func (l *LogServer) KillSwitch() chan bool {
	return l.quitChan
}

// Quit stops the server and all goroutines
func (l *LogServer) Quit() {

	for _, quitChan := range l.killswitches {
		quitChan <- true
	}

	if err := l.listenUnix.Close(); err != nil {
		fmt.Printf("Quit: could not close unix-socket listener: %s\n", err.Error())
	}

	if err := l.listenTCP.Close(); err != nil {
		fmt.Printf("Quit: could not close tcp-socket listener: %s\n", err.Error())
	}

	l.wg.Wait()
}

// Config contains all the configuration for the remote logger
type Config struct {

	// Remote logger config
	Host         string
	Port         int
	UnixSockPath string
	TokenPath    string

	// Local logger config
	LoggerConfig *log.Config
}

// New creates a new logserver instance
func New(config *Config) (*LogServer, error) {

	// Instantiate remote logserver
	rLogger := &LogServer{}

	// Listen on to the unix socket
	listenUnix, err := net.Listen("unix", config.UnixSockPath)
	if err != nil {
		return nil, fmt.Errorf("New: could not listen on the unix socket: %s", err.Error())
	}

	// Serve socket requests
	once := &sync.Once{}
	ready, quitChan, connChan := make(chan bool, 1), make(chan bool, 1), make(chan net.Conn, 1)
	rLogger.killswitches = append(rLogger.killswitches, quitChan)
	go func() {
	Loop:
		for {

			// Listen for incoming unix connections
			go func() {
				fd, errUnix := listenUnix.Accept()
				if errUnix != nil {
					return
				}
				connChan <- fd
			}()

			// Announce readyness
			once.Do(func() { ready <- true })

			// Handle connections or quit
			select {
			case conn := <-connChan:
				go rLogger.HandleUnixRequest(conn)
			case <-quitChan:
				break Loop
			}
		}
	}()
	<-ready

	// Listen on tcp
	listenTCP, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		listenUnix.Close()
		return nil, fmt.Errorf("New: could not listen on tcp socket: %s", err.Error())
	}

	// Create Auth interceptor
	intercept := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if errAuth := rLogger.Authorize(ctx); errAuth != nil {
			return nil, errAuth
		}
		return handler(ctx, req)
	}

	// Put everything together
	rLogger.unixSockPath = config.UnixSockPath
	rLogger.listenUnix = listenUnix
	rLogger.listenTCP = listenTCP
	rLogger.server = grpc.NewServer(grpc.UnaryInterceptor(intercept))
	rLogger.quitChan = make(chan bool, 1)

	// Serve gRPC requests
	logrpc.RegisterRemoteLoggerServer(rLogger.server, rLogger)
	quitChan, failChan := make(chan bool, 1), make(chan error, 1)
	rLogger.killswitches = append(rLogger.killswitches, quitChan)
	go func() {
		if errTCP := rLogger.server.Serve(listenTCP); errTCP != nil {
			failChan <- errTCP
		}
	}()

	// Quit if gRPC server fails
	go func() {
		select {
		case errTCP := <-failChan:
			if errTCP != nil {
				fmt.Printf("New: could not serve TCP requests: %s\n", errTCP.Error())
				rLogger.Quit()
				os.Exit(1)
			}
		case <-time.After(10 * time.Second):
		}
	}()

	// Wait for gRPC server to start up
	//rLogger.wg.Add(1)
	go func() {
		<-quitChan
		rLogger.server.Stop()
		//rLogger.wg.Done()
	}()

	// Instantiate logger
	logger, err := log.New(config.LoggerConfig)
	if err != nil {
		return nil, fmt.Errorf("New: could not start logger: %s", err.Error())
	}
	rLogger.logger = logger

	return rLogger, nil
}
