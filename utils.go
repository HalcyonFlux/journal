package log

import (
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// getMsgCode returns message code's string type
func (l *Logger) getMsgCode(code int) (string, bool) {

	resp, ok := l.codes[code]
	if !ok {
		return "UNKOWN", true
	}
	return resp.Type, resp.Error
}

// rotateFile creates a new and archives the old logfile
func (l *Logger) rotateFile() killswitch {
	quitChan := make(chan bool, 1)

	// Prepare stdout
	if l.config.Out == OUT_STDOUT {
		l.stdout = os.Stdout
		return quitChan
	}
	if l.config.Out == OUT_FILE_AND_STDOUT {
		l.stdout = os.Stdout
	}

	// Start the rotation coroutine
	ready := make(chan bool, 1)
	go func() {
		prev := ""
		current := rotationDate(l.config.Rotation, 0)
		next := rotationDate(l.config.Rotation, 1)

		// Compress old files (if not yet done so)
		if l.config.Compress {
			compressOld(l.config.Folder, fmt.Sprintf("%s_%s", l.config.Filename, current))
		}

		var once sync.Once
	Loop:
		for {

			if current = time.Now().Format("2006-01-02"); prev == "" || (current != prev && current == next) {

				// Update relevant dates
				next = rotationDate(l.config.Rotation, 1)
				d1, _ := time.Parse("2006-01-02", next)
				d2, _ := time.Parse("2006-01-02", current)
				delta := d1.Unix() - d2.Unix() - 60

				// Open the new logfile
				newLogfile := fmt.Sprintf("%s/%s_%s.log", l.config.Folder, l.config.Filename, current)
				isNew := false
				if _, err := os.Stat(newLogfile); os.IsNotExist(err) {
					isNew = true
				}

				f, err := os.OpenFile(newLogfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
				if err != nil {
					l.Log("system", 1, "rotateFile could not open a new logfile: %s", err.Error())
					continue
				}

				// Replace local writers
				l.mu.Lock()
				l.logfile.Close()
				l.logfile = f
				if isNew && !l.config.JSON {
					l.logfile.WriteString(fmt.Sprintf("%s\n", l.headers()))
				}
				l.mu.Unlock()

				// Compress and delete old file
				if l.config.Compress && prev != "" {
					if err := compress(l.config.Folder, fmt.Sprintf("%s_%s", l.config.Filename, prev)); err != nil {
						l.Log("rotateFile", 1, "Could not compress old logfile: %s", err.Error())
					}
				}

				// Update previous date
				prev = current

				// Proceed with main routine
				once.Do(func() { ready <- true })

				// Wait for up until one minute before the next date
				select {
				case <-time.After(time.Duration(delta) * time.Second):
				case <-quitChan:
					break Loop
				}

			}

			// Wait for a second
			select {
			case <-time.After(1 * time.Second):
			case <-quitChan:
				break Loop
			}

		}
	}()

	<-ready
	return quitChan
}

// rotationDate returns a log's rotation date with a specific offset
// , e.g.: 0 - current, 1 - next, -1 - previous.
func rotationDate(rotation int, offset int) string {
	suffix := time.Now().Format("2006-01-02")

	switch rotation {
	case ROT_DAILY:
		shift := time.Now().AddDate(0, 0, offset)
		suffix = fmt.Sprintf("%s", shift.Format("2006-01-02"))
	case ROT_WEEKLY:
		shift := time.Now().AddDate(0, 0, offset*7)
		if day := int(shift.Weekday()); day == 0 {
			suffix = fmt.Sprintf("%s", shift.AddDate(0, 0, -6).Format("2006-01-02"))
		} else {
			suffix = fmt.Sprintf("%s", shift.AddDate(0, 0, -(day-1)).Format("2006-01-02"))
		}
	case ROT_MONTHLY:
		shift := time.Now().AddDate(0, 1, 0)
		suffix = fmt.Sprintf("%s-01", shift.Format("2006-01"))
	case ROT_ANNUALLY:
		shift := time.Now().AddDate(1, 0, 0)
		suffix = fmt.Sprintf("%s-01-01", shift.Format("2006"))
	}

	return suffix
}

// compress compresses a logfile and deletes the old one
func compress(folder, file string) error {

	// Relevant files
	filepath := fmt.Sprintf("%s/%s.log", folder, file)
	gzipfilepath := fmt.Sprintf("%s/%s.log.gz", folder, file)

	// Open logfile
	// (fails if file does not exist)
	f, err := os.OpenFile(filepath, os.O_RDONLY, 0600)
	if err != nil {
		return fmt.Errorf("compress: could not open logfile: %s", err.Error())
	}

	// Open gzipfile
	fzip, err := os.OpenFile(gzipfilepath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("compress: could not open archive file: %s", err.Error())
	}

	// gzip writer and metadata
	zip, err := gzip.NewWriterLevel(fzip, flate.BestCompression)
	if err != nil {
		return fmt.Errorf("compress: could not create gzip writer: %s", err.Error())
	}
	zip.Name = fmt.Sprintf("%s.log", file)
	zip.Comment = "Archive logfile"
	zip.ModTime = time.Now().UTC()

	// Read and zip contents
	buf := make([]byte, 4<<20)
	for {

		n, err := f.Read(buf)
		if n == 0 {
			if err != nil && err != io.EOF {
				return fmt.Errorf("compress: could not read log: %s", err.Error())
			} else if err == io.EOF {
				break
			}
		}

		if _, err := zip.Write(buf[:n]); err != nil {
			return fmt.Errorf("compress: could not archive log: %s", err.Error())
		}
	}

	// Close zip writer
	if err := zip.Close(); err != nil {
		return fmt.Errorf("compress: could not close archive writer: %s", err.Error())
	}

	// Sync zip file
	if err := fzip.Sync(); err != nil {
		return fmt.Errorf("compress: could not sync archive file: %s", err.Error())
	}

	// Close zip file
	if err := fzip.Close(); err != nil {
		return fmt.Errorf("compress: could not close archive file: %s", err.Error())
	}

	// Close logfile
	if err := f.Close(); err != nil {
		return fmt.Errorf("compress: could not close log file: %s", err.Error())
	}

	// Remove logfile
	if err := os.RemoveAll(filepath); err != nil {
		return fmt.Errorf("compress: could not delete old logfile: %s", err.Error())
	}

	return nil
}

// compressOld compresses all logfiles except one (current)
func compressOld(folder, except string) {

	files, _ := ioutil.ReadDir(folder)
	for _, f := range files {
		if !f.IsDir() && path.Ext(f.Name()) == ".log" && f.Name() != fmt.Sprintf("%s.log", except) {
			compress(folder, strings.TrimSuffix(f.Name(), ".log"))
		}
	}

}

// headers returns log's column headers as a tab-separated string
func (l *Logger) headers() string {
	header := make([]string, len(l.config.Columns))
	for i, code := range l.config.Columns {
		header[i] = colname(code)
	}

	return strings.Join(header, "\t")
}

// pushToLedger pushes a log entry into the ledger
func (l *Logger) pushToLedger(depth int, caller string, code int, msg string, format ...interface{}) error {

	// An active Logger will wait for the transit to finish
	inTransit := l.active
	if inTransit {
		l.wg.Add(1)
	}

	// Format message
	fmsg := msg
	if len(format) > 0 {
		fmsg = fmt.Sprintf(msg, format...)
	}

	// Get some additional information
	_, file, line, _ := runtime.Caller(depth)
	name, isErr := l.getMsgCode(code)

	// Prepare log entry
	entry := logEntry{}
	for i := int64(COL_DATE_YYMMDD); i <= int64(COL_LINE); i++ {
		switch i {
		case COL_DATE_YYMMDD:
			entry[i] = time.Now().Format("2006-01-02")
		case COL_DATE_YYMMDD_HHMMSS:
			entry[i] = time.Now().Format("2006-01-02 15:04:05")
		case COL_DATE_YYMMDD_HHMMSS_NANO:
			entry[i] = time.Now().Format("2006-01-02 15:04:05.000000000")
		case COL_TIMESTAMP:
			entry[i] = strconv.FormatInt(time.Now().Unix(), 10)
		case COL_SERVICE:
			entry[i] = l.config.Service
		case COL_INSTANCE:
			entry[i] = l.config.Instance
		case COL_CALLER:
			entry[i] = caller
		case COL_MSG_TYPE_SHORT:
			if isErr {
				entry[i] = "ERR"
			} else {
				entry[i] = "MSG"
			}
		case COL_MSG_TYPE_INT:
			entry[i] = strconv.Itoa(code)
		case COL_MSG_TYPE_STR:
			entry[i] = name
		case COL_MSG:
			entry[i] = fmsg
		case COL_FILE:
			entry[i] = file
		case COL_LINE:
			entry[i] = strconv.Itoa(line)
		}
	}

	// Write entry into the ledger
	if inTransit {
		go func() {
			l.ledger <- entry
		}()
	}

	// Return error
	if isErr {
		return fmt.Errorf("%s", fmsg)
	}

	return nil
}

// write processes the log ledger and writes entries to all the relevant sources
// (local file, stdout, remote file, kafka)
func (l *Logger) write() killswitch {
	quitChan := make(chan bool, 1)

	ready := make(chan bool, 1)
	go func() {

		var once sync.Once
	Loop:
		for {
			once.Do(func() { ready <- true })

			select {
			case entry := <-l.ledger:

				l.mu.Lock()

				// Write to stdout
				if l.stdout != nil {
					l.stdout.WriteString(fmt.Sprintf("%s\n", entry.toStr(l.config.Columns)))
				}

				// Write to local file
				if l.logfile != nil {
					if l.config.JSON {
						l.logfile.WriteString(fmt.Sprintf("%s\n", entry.toJSON(l.config.Columns)))
					} else {
						l.logfile.WriteString(fmt.Sprintf("%s\n", entry.toStr(l.config.Columns)))
					}
				}

				// Write to remote backends
				if len(l.remoteWriters) > 0 {
					jsoned, err := json.Marshal(entry)
					if err != nil {
						l.Log("system", 1, "write: could not marshal log entry: %s", err.Error())
					}

					for _, remote := range l.remoteWriters {
						if _, err := remote.Write(jsoned); err != nil {
							l.Log("system", 1, "write: could not send log to a remote backend: %s", err.Error())
						}
					}
				}

				l.wg.Done()
				l.mu.Unlock()

			case <-quitChan:
				break Loop
			}

		}
	}()

	<-ready
	return quitChan
}

// canWrite checks if the directory is writeable
func canWrite(folder string) bool {

	f, err := ioutil.TempFile(folder, "write_test")
	if err != nil {
		return false
	}

	name := f.Name()
	f.Close()
	os.Remove(name)

	return true
}
