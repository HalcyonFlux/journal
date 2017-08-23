package log

import (
	"encoding/json"
	"fmt"
	"github.com/vaitekunas/log/logrpc"
	"io"
	"os"
	"sync"
	"time"
)

// Config contains all the necessary settings to build a new logging facility
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

// Killswitch is a bool channel used to stop coroutines
type killswitch chan<- bool

// New creates a new logging facility
func New(config *Config) (*Logger, error) {

	// Validate options
	if config.Rotation < ROT_CONTINUOUS || config.Rotation > ROT_ANNUALLY {
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

	// Initiate log instance
	Log := &Logger{
		Mutex:            &sync.Mutex{},
		active:           true,
		config:           config,
		codes:            defaultCodes,
		ledger:           make(chan logEntry, 1000),
		ledgerTransit:    &sync.WaitGroup{},
		ledgerProcessing: &sync.WaitGroup{},
	}

	// Start file rotation (async)
	Log.killswitches = append(Log.killswitches, Log.rotateFile())

	// Start log writer
	Log.killswitches = append(Log.killswitches, Log.write())

	return Log, nil
}

// Logger is the main loggger struct
type Logger struct {
	*sync.Mutex
	active bool         // logger Activity switch
	config *Config      // Main config
	codes  map[int]Code // Mapping of integer message codes to their string values

	ledger           chan logEntry   // Ledger of unprocessed log entries
	ledgerTransit    *sync.WaitGroup // Waitgroup for messages in transit (being sent to the ledger)
	ledgerProcessing *sync.WaitGroup // Waitgroup for messages in process (being written to various backends)
	killswitches     []killswitch    // Killswitches of all coroutines spawned by the logger

	// log Writers
	logfile       *os.File    // local logfile's file descriptor
	stdout        *os.File    // local stdout
	remoteWriters []io.Writer // remote log writers (grpc, kafka, etc)

	// gRPC-related
	gRPC        *logrpc.RemoteLoggerClient // gRPC client
	gRPCTimeout time.Duration              // gRPC timeout duration
}

// UseCustomCodes Replaces loggers default message codes with custom ones
func (l *Logger) UseCustomCodes(codes map[int]Code) {
	for code, lCode := range codes {
		if code > 1 && code < 999 {
			l.codes[code] = lCode
		}
	}
}

// Log logs a simple message and returns nil or error, depending on the code
func (l *Logger) Log(caller string, code int, msg string, format ...interface{}) error {
	return l.pushToLedger(2, caller, code, msg, format...)
}

// LogFields encodes the message (not the whole log) in JSON and writes to log
func (l *Logger) LogFields(caller string, code int, msg map[string]interface{}) error {
	jsoned, err := json.Marshal(msg)
	if err != nil {
		return l.pushToLedger(2, "system", 1, "LogFields: could not marshal log entry to JSON: %s", err.Error())
	}

	return l.pushToLedger(2, caller, code, string(jsoned))
}

// NewCaller is a wrapper for the Logger.Log function
func (l *Logger) NewCaller(caller string) func(int, string, ...interface{}) error {

	return func(code int, msg string, format ...interface{}) error {
		return l.pushToLedger(2, caller, code, msg, format...)
	}

}

// NewCallerCode is a wrapper for the Logger.fullog function
func (l *Logger) NewCallerCode(caller string, code int) func(string, ...interface{}) error {

	return func(msg string, format ...interface{}) error {
		return l.pushToLedger(2, caller, code, msg, format...)
	}

}

// AddRemote adds a remote backend to send logs to
func (l *Logger) AddRemote(writer io.Writer) {
	l.remoteWriters = append(l.remoteWriters, writer)
}

// Quit stops all Logger coroutines and closes files
func (l *Logger) Quit() {

	// Deactivate ledger
	l.active = false

	// Wait for the ledger transits to finish
	l.ledgerTransit.Wait()

	// Wait for the ledger processing to finish
	l.ledgerProcessing.Wait()

	// Lock any writing or file rotation activity
	l.Lock()
	defer l.Unlock()

	// Stop all registered coroutines
	for _, killswitch := range l.killswitches {
		killswitch <- true
	}

	// Close active log
	if l.logfile != nil {
		l.logfile.Close()
	}

}
