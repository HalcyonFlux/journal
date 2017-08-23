package log

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// Log entry correction pattern
var correctionPattern = regexp.MustCompile("[\t\n\r\b\f\v]")

// logEntry contains all the column values of a log entry
type logEntry map[int64]string // Compatible with logrpc.LogEntry.Entry

// correct corrects some possible mistakes in logEntry
func (l logEntry) correct() {

	for i, v := range l {
		if v == "" {
			l[i] = "N/A"
		}
		l[i] = correctionPattern.ReplaceAllString(l[i], " ")
	}

}

// toStr turns logEntry to string
func (l logEntry) toStr(cols []int64) string {
	msg := ""
	for _, code := range cols {
		msg = fmt.Sprintf("%s%s\t", msg, l[code])
	}
	return msg
}

// toJSON turns logEntry to json-encoded string
func (l logEntry) toJSON(cols []int64) string {
	nameLog := map[string]string{}
	for _, code := range cols {
		nameLog[colname(code)] = l[code]
	}

	jsoned, err := json.Marshal(nameLog)
	if err != nil {

	}
	return string(jsoned)
}
