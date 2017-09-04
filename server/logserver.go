package server

import (
	"fmt"
	"github.com/vaitekunas/journal"
	"github.com/vaitekunas/journal/logrpc"
	"net"
	"os"
	"sync"
	"time"

	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// killswitch is used to close all goroutines
type killswitch chan<- bool

// Config contains all the configuration for the remote logger
type Config struct {

	// Remote logger config
	Host         string
	Port         int
	UnixSockPath string
	TokenPath    string
	StatsPath    string

	// Local logger config
	LoggerConfig *journal.Config
}

// New creates a new logserver instance
func New(config *Config) (*LogServer, error) {

	// Instantiate remote logserver
	rLogger := &LogServer{Mutex: &sync.Mutex{}}

	// Internal context used to cancel supporting goroutines
	internalCTX, cancel := context.WithCancel(context.Background())

	// Listen on to the unix socket
	listenUnix, err := net.Listen("unix", config.UnixSockPath)
	if err != nil {
		return nil, fmt.Errorf("New: could not listen on the unix socket: %s", err.Error())
	}

	// Serve socket requests
	connChan := make(chan net.Conn, 1)

	// Listen for incoming unix connections
	go func() {
	Loop:
		for {
			fd, errUnix := listenUnix.Accept()
			if errUnix != nil {
				continue
			}
			select {
			case connChan <- fd:
			case <-internalCTX.Done():
				break Loop
			}
		}
	}()

	// Process unix connections
	go func() {
	Loop:
		for {
			select {
			case conn := <-connChan:
				go rLogger.HandleUnixRequest(conn)
			case <-internalCTX.Done():
				break Loop
			}
		}
	}()

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
	rLogger.cancelSupport = cancel
	rLogger.unixSockPath = config.UnixSockPath
	rLogger.listenUnix = listenUnix
	rLogger.listenTCP = listenTCP
	rLogger.statsPath = config.StatsPath
	rLogger.tokenPath = config.TokenPath
	rLogger.server = grpc.NewServer(grpc.UnaryInterceptor(intercept))
	rLogger.stats = make(map[string]*Statistic)
	rLogger.tokens = make(map[string]string)
	rLogger.quitChan = make(chan bool, 1)

	// Load auth tokens from disk
	if errToken := rLogger.loadTokensFromDisk(); errToken != nil {
		return nil, fmt.Errorf("New: could not load tokens from disk: %s", errToken.Error())
	}

	// Load statistics from disk
	if errStats := rLogger.loadStatisticsFromDisk(); errStats != nil {
		return nil, fmt.Errorf("New: could not load statistics from disk: %s", errStats.Error())
	}

	// Periodically dump statistics to file
	go rLogger.periodicallyDumpStats(internalCTX, 60*time.Second)

	// Serve gRPC requests
	logrpc.RegisterRemoteLoggerServer(rLogger.server, rLogger)
	failChan := make(chan error, 1)
	go func() {
		if errTCP := rLogger.server.Serve(listenTCP); errTCP != nil {
			failChan <- errTCP
		}
	}()

	// Quit if gRPC server fails (wait for 60 seconds to be sure)
	go func() {
		select {
		case errTCP := <-failChan:
			if errTCP != nil {
				fmt.Printf("New: could not serve TCP requests: %s\n", errTCP.Error())
				rLogger.Quit()
				os.Exit(1)
			}
		case <-time.After(60 * time.Second):
		}
	}()

	// Wait for gRPC server to start up
	go func() {
		<-internalCTX.Done()
		rLogger.server.Stop()
	}()

	// Instantiate logger
	logger, err := journal.New(config.LoggerConfig)
	if err != nil {
		return nil, fmt.Errorf("New: could not start logger: %s", err.Error())
	}
	rLogger.logger = logger

	return rLogger, nil
}

// Statistic contains various log-related statistics
type Statistic struct {
	Service         string
	Instance        string
	LogsParsed      [24]int64
	LogsParsedBytes [24]int64
	LastIP          string
	LastActive      time.Time
}

// LogServer implements log.Logger and log.RemoteLoggerServer interfaces
type LogServer struct {
	*sync.Mutex // Mutex for tokens and statistics

	logger *journal.Logger // Local logger
	server *grpc.Server    // gRPC server

	unixSockPath string       // Path to the unix socket file
	listenUnix   net.Listener // Unix-socket listener (unix)
	listenTCP    net.Listener // TCP listener (grpc)

	cancelSupport func() // Internal context cancel function to stop all supporting goroutines

	statsPath string                // A path to the file where all the statistics are kept
	stats     map[string]*Statistic // Log statistics map[service/instance]*Statistic

	tokenPath string            // A path to the file where all the tokens are kept
	tokens    map[string]string // Authorization tokens map[service/instance]token

	quitChan chan bool // Internal kill switch
}

// RemoteLog handles incoming remote logs
func (l *LogServer) RemoteLog(ctx context.Context, logEntry *logrpc.LogEntry) (*logrpc.Nothing, error) {

	// Extract credentials
	service, instance, key, _, err := extractCaller(ctx)
	if err != nil {
		return nil, fmt.Errorf("RemoteLog: could not extract caller credentials")
	}

	// Update statistics
	go l.GatherStatistics(service, instance, key, logEntry)

	// Push entry into the log entry channel
	if err := l.logger.RawEntry(logEntry.GetEntry()); err != nil {
		return nil, fmt.Errorf("RemoteLog: could not process raw log: %s", err.Error())
	}

	return &logrpc.Nothing{}, nil
}

// Authorize is a gRPC interceptor that authorizes incoming RPCs
func (l *LogServer) Authorize(ctx context.Context) error {

	// Verify presence of metadata
	_, _, key, token, err := extractCaller(ctx)
	if err != nil {
		return fmt.Errorf("Authorize: cannot extract caller credentials :%s", err.Error())
	}

	// Get existing token
	realToken, ok := l.tokens[key]
	if !ok {
		return fmt.Errorf("Authorize: unknown service/instance")
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

	// Stop all supporting goroutines
	l.cancelSupport()

	// Close unix listener
	if err := l.listenUnix.Close(); err != nil {
		fmt.Printf("Quit: could not close unix-socket listener: %s\n", err.Error())
	}

	// Close TCP listener
	if err := l.listenTCP.Close(); err != nil {
		fmt.Printf("Quit: could not close tcp-socket listener: %s\n", err.Error())
	}
}
