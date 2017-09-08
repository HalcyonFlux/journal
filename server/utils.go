package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	context "golang.org/x/net/context"
	metadata "google.golang.org/grpc/metadata"
)

// Extracts service, instance and token from the grpc context
func extractCaller(ctx context.Context) (service, instance, key, token, ip string, err error) {

	// Verify presence of metadata
	md, ok := metadata.FromContext(ctx)
	if !ok {
		return "", "", "", "", "", fmt.Errorf("Authorize: missing metadata")
	}

	// Verify that all required items are available
	for _, key := range []string{"service", "instance", "token", "ip"} {
		if slice, okKey := md[key]; !okKey || len(slice) != 1 {
			return "", "", "", "", "", fmt.Errorf("Authorize: missing %s", key)
		}
	}

	// Extract the real token
	service = md["service"][0]
	instance = md["instance"][0]
	key = fmt.Sprintf("%s/%s", strings.ToLower(service), strings.ToLower(instance))
	token = md["token"][0]
	ip = md["ip"][0]

	return service, instance, key, token, ip, nil
}

// Verifies that a file exist
func fileExists(filename string) error {

	// File dir
	dirPath := filepath.Dir(filename)

	// Make sure dir and file exist
	if dir, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dirPath, 0700); err != nil {
			return fmt.Errorf("fileExists: directory to store tokens.db could not be created: %s", err.Error())
		}
	} else if !dir.IsDir() {
		return fmt.Errorf("fileExists: token path is not a directory")
	}

	// Make sure the file exists
	if d, err := os.Stat(filename); os.IsNotExist(err) {
		f, errF := os.Create(filename)
		if errF != nil {
			return fmt.Errorf("fileExists: could not create token db: %s", err.Error())
		}
		return f.Close()
	} else if d.IsDir() {
		return fmt.Errorf("fileExists: no filename provided?")
	}

	return nil
}

// getCleanKey cleans inputs and builds from them a service/instance key
func getCleanKey(service, instance string) string {
	return strings.ToLower(fmt.Sprintf("%s/%s", strings.TrimSpace(service), strings.TrimSpace(instance)))
}

// bold returns a bolded version of v
func bold(v interface{}) interface{} {
	return color.New(color.Bold).Sprint(v)
}

// console writes a message with a timestamp to console
func console(s interface{}) string {
	return fmt.Sprintf(" %s [%s] %v", color.New(color.FgHiBlue).Sprint("▶"), time.Now().Format("2006-01-02 15:04:05"), s)
}

// parsedSums sums and formats parsed log statistics
func parsedSums(parsedLogs, parsedBytes [24]int64) (string, string, int64, int64) {
	var plogs int64
	var pbytes int64

	for i := 0; i < 24; i++ {
		plogs += parsedLogs[i]
		pbytes += parsedBytes[i]
	}

	// Add thousands separator for parsed logs
	plogsStr := strconv.FormatInt(plogs, 10)
	seps := int(len(plogsStr)/3) + 1
	plogsTsd := make([]string, seps)
	for i := 1; i <= seps; i++ {
		if len(plogsStr)-i*3 >= 0 {
			plogsTsd[seps-i] = plogsStr[len(plogsStr)-i*3 : len(plogsStr)-(i-1)*3]
		} else {
			plogsTsd[seps-i] = plogsStr[:len(plogsStr)-(i-1)*3]
		}
	}

	// Normalize parsed bytes
	var pbytesNorm float64
	var pbytesSuffix string
	if div := int64(1E3); pbytes <= div {
		pbytesNorm = float64(pbytes)
		pbytesSuffix = "B"
	} else if div := int64(1E6); pbytes <= div {
		pbytesNorm = float64(pbytes) / float64(div/1E3)
		pbytesSuffix = "kB"
	} else if div := int64(1E9); pbytes <= div {
		pbytesNorm = float64(pbytes) / float64(div/1E3)
		pbytesSuffix = "MB"
	} else if div := int64(1E12); pbytes <= div {
		pbytesNorm = float64(pbytes) / float64(div/1E3)
		pbytesSuffix = "GB"
	} else if div := int64(1E15); pbytes <= div {
		pbytesNorm = float64(pbytes) / float64(div/1E3)
		pbytesSuffix = "TB"
	} else if div := int64(1E18); pbytes <= div {
		pbytesNorm = float64(pbytes) / float64(div/1E3)
		pbytesSuffix = "PB"
	}

	return strings.TrimSpace(strings.Join(plogsTsd, ".")),
		strings.TrimSpace(fmt.Sprintf("%6.2f %s", pbytesNorm, pbytesSuffix)),
		plogs,
		pbytes

}
