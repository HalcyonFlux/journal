package journal

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/vaitekunas/journal/logrpc"
	"golang.org/x/net/context"
)

// Config contains all the necessary settings to create a new local logging facility
type Config struct {
	Service  string  // Service name
	Instance string  // Instance name
	Folder   string  // Folder to store logfiles (can be empty if logging to stdout only)
	Filename string  // Filename of the logfiles (without date suffix and file extension. Can be empty if logging to stdout only)
	Rotation int     // Logfile rotation frequency
	Out      int     // Logger output type
	Headers  bool    // Should the logfile contain column headers?
	JSON     bool    // Should each entry be written as a JSON-formatted string?
	Compress bool    // Should old logfiles be compressed?
	Columns  []int64 // List of relevant columns (can be empty if default columns should be used)
}

// New creates a new logging facility
func New(config *Config) (Logger, error) {

	// Validate options
	if config.Rotation < ROT_NONE || config.Rotation > ROT_ANNUALLY {
		return nil, fmt.Errorf("New: invalid roll option '%d'", config.Rotation)
	}
	if config.Out < OUT_FILE || config.Out > OUT_FILE_AND_STDOUT {
		return nil, fmt.Errorf("New: invalid output option '%d'", config.Out)
	}

	if len(config.Columns) == 0 {
		config.Columns = defaultCols
	} else {
		for _, col := range config.Columns {
			if col < COL_DATE_YYMMDD || col > COL_LINE {
				return nil, fmt.Errorf("New: invalid column '%d'", col)
			}
		}
	}

	// Check permissions
	if config.Out == OUT_FILE || config.Out == OUT_FILE_AND_STDOUT {
		if !canWrite(config.Folder) {
			return nil, fmt.Errorf("New: cannot write to '%s'", config.Folder)
		}
	}

	// Internal context
	internalCTX, cancel := context.WithCancel(context.Background())

	// Initiate log instance
	Log := &logger{
		mu:            &sync.Mutex{},
		wg:            &sync.WaitGroup{},
		active:        true,
		config:        config,
		codes:         defaultCodes,
		ledger:        make(chan logEntry, 1000),
		remoteWriters: map[string]io.Writer{},
		cancel:        cancel,
	}

	// Start file rotation (async)
	Log.rotateFile(internalCTX)

	// Start log writer
	Log.write(internalCTX)

	return Log, nil
}

// logger is the main loggger struct
type logger struct {
	mu *sync.Mutex     // Protect logfile changes
	wg *sync.WaitGroup // Protect ledger processing

	active bool         // logger Activity switch
	config *Config      // Main config
	codes  map[int]Code // Mapping of integer message codes to their string values

	ledger chan logEntry // Ledger of unprocessed log entries
	cancel func()        // Function to cancel internal  context

	// log Writers
	logfile       *os.File             // local logfile's file descriptor
	stdout        *os.File             // local stdout
	remoteWriters map[string]io.Writer // remote log writers (grpc, kafka, etc)

	// gRPC-related
	gRPC        *logrpc.RemoteLoggerClient // gRPC client
	gRPCTimeout time.Duration              // gRPC timeout duration
}

// UseCustomCodes Replaces loggers default message codes with custom ones
func (l *logger) UseCustomCodes(codes map[int]Code) {
	for code, lCode := range codes {
		if code > 1 && code < 999 {
			l.codes[code] = lCode
		}
	}
}

// Log logs a simple message and returns nil or error, depending on the code
func (l *logger) Log(caller string, code int, msg string, format ...interface{}) error {
	return l.pushToLedger(2, caller, code, msg, format...)
}

// LogFields encodes the message (not the whole log) in JSON and writes to log
func (l *logger) LogFields(caller string, code int, msg map[string]interface{}) error {
	jsoned, err := json.Marshal(msg)
	if err != nil {
		return l.pushToLedger(2, "system", 1, "LogFields: could not marshal log entry to JSON: %s", err.Error())
	}

	return l.pushToLedger(2, caller, code, string(jsoned))
}

// NewCaller is a wrapper for the Logger.Log function
func (l *logger) NewCaller(caller string) func(int, string, ...interface{}) error {

	return func(code int, msg string, format ...interface{}) error {
		return l.pushToLedger(2, caller, code, msg, format...)
	}

}

// NewCallerWithFields is a wrapper for the Logger.LogFields function
func (l *logger) NewCallerWithFields(caller string) func(int, map[string]interface{}) error {

	return func(code int, msg map[string]interface{}) error {
		return l.LogFields(caller, code, msg)
	}

}

// RawEntry writes a raw log entry (map of strings) into the ledger.
// The raw entry must contain columns COL_DATE_YYMMDD_HHMMSS_NANO to COL_LINE
func (l *logger) RawEntry(entry map[int64]string) error {

	// Validate the raw Entry
	for _, code := range defaultCols {
		if _, ok := entry[code]; !ok {
			return fmt.Errorf("RawEntry: missing column '%d'", code)
		}
	}

	// Write the entry into the ledger
	if l.active {
		l.wg.Add(1)
		go func() {
			l.ledger <- entry
		}()
	}

	return nil
}

// AddDestination adds a (remote) destination to send logs to
func (l *logger) AddDestination(name string, writer io.Writer) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.remoteWriters[name]; ok {
		return fmt.Errorf("AddDestination: destination %s already present", name)
	}

	l.remoteWriters[name] = writer

	return nil
}

// RemoveDestination removes a (remote) destination to send logs to
func (l *logger) RemoveDestination(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.remoteWriters[name]; !ok {
		return fmt.Errorf("RemoveDestination: unknown destination '%s'", name)
	}

	delete(l.remoteWriters, name)

	return nil
}

// ListDestinations lists all (remote) destinations
func (l *logger) ListDestinations() []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	var localDst []string

	switch l.config.Out {
	case OUT_STDOUT:
		localDst = []string{"stdout"}
	case OUT_FILE:
		localDst = []string{l.logfile.Name()}
	case OUT_FILE_AND_STDOUT:
		localDst = []string{"stdout", l.logfile.Name()}
	}

	remoteDst := make([]string, len(l.remoteWriters))
	i := 0
	for endpoint := range l.remoteWriters {
		remoteDst[i] = endpoint
		i++
	}
	sort.Strings(remoteDst)

	return append(localDst, remoteDst...)
}

// Quit stops all Logger coroutines and closes files
func (l *logger) Quit() {

	// Deactivate ledger
	l.active = false

	// Wait for the ledger processing to finish
	l.wg.Wait()

	// Lock any writing or file rotation activity
	l.mu.Lock()
	defer l.mu.Unlock()

	// Stop all registered goroutines
	l.cancel()

	// Close active log
	if l.logfile != nil {
		l.logfile.Close()
	}

}
