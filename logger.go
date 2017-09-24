package journal

import (
  "io"
)

// Logger is the main interface implemented by journal
type Logger interface{

    // AddDestination adds a (remote) destination to send logs to
    AddDestination(name string, writer io.Writer) error

    // ListDestinations lists all (remote) destinations
    ListDestinations() []string

    // Log logs a simple message and returns nil or error, depending on the code
    Log(caller string, code int, msg string, format ...interface{}) error

    // LogFields encodes the message (not the whole log) in JSON and writes to lo
    LogFields(caller string, code int, msg map[string]interface{}) error

    // NewCaller is a wrapper for the Logger.Log function
    NewCaller(caller string) func(int, string, ...interface{}) error

    // NewCallerWithFields is a wrapper for the Logger.LogFields function
    NewCallerWithFields(caller string) func(int, map[string]interface{}) error

    // Quit stops all Logger coroutines and closes files
    Quit()

    // RawEntry writes a raw log entry (map of strings) into the ledger. The raw entry must contain columns COL_DATE_YYMMDD_HHMMSS_NANO to COL_LINE
    RawEntry(entry map[int64]string) error

    // RemoveDestination removes a (remote) destination to send logs to
    RemoveDestination(name string) error

    // UseCustomCodes Replaces loggers default message codes with custom ones
    UseCustomCodes(codes map[int]Code)

}
