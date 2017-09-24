package server

import (
	"fmt"
	"io/ioutil"
)

// Logfiles returns statistics about available log files
func (l *logServer) Logfiles() (map[string]string, error) {
	files, err := ioutil.ReadDir(l.logfolder)
	if err != nil {
		return nil, fmt.Errorf("Logfiles: could not list logfiles: %s", err.Error())
	}

	logs := make(map[string]string, len(files))

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		size := file.Size()
		_, pbytesStr := prettyParsedSums(0, size)

		logs[name] = pbytesStr
	}
	return logs, nil
}
