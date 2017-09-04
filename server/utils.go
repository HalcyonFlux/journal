package server

import (
	"fmt"
	"github.com/fatih/color"
	context "golang.org/x/net/context"
	metadata "google.golang.org/grpc/metadata"
	"os"
	"path/filepath"
	"strings"
)

// Extracts service, instance and token from the grpc context
func extractCaller(ctx context.Context) (service, instance, key, token string, err error) {

	// Verify presence of metadata
	md, ok := metadata.FromContext(ctx)
	if !ok {
		return "", "", "", "", fmt.Errorf("Authorize: missing metadata")
	}

	// Verify that all required items are available
	for _, key := range []string{"service", "instance", "token"} {
		if slice, okKey := md[key]; !okKey || len(slice) != 1 {
			return "", "", "", "", fmt.Errorf("Authorize: missing %s", key)
		}
	}

	// Extract the real token
	service = md["service"][0]
	instance = md["instance"][0]
	key = fmt.Sprintf("%s/%s", strings.ToLower(service), strings.ToLower(instance))
	token = md["token"][0]

	return service, instance, key, token, nil
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
