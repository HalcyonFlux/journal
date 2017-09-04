package journal

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

// Test/example setup
func setup(t *testing.T) (tempdir string, teardown func()) {

	dir, err := ioutil.TempDir(os.Getenv("HOME"), "example")
	if err != nil {
		t.Errorf("Could not create tempdir: %s", err.Error())
	}

	return dir, func() {
		os.RemoveAll(dir)
	}

}

// Simple example of local logging
func ExampleLogger_Log() {

	// Create temporary folder and teardown function
	tempdir, teardown := setup(&testing.T{})
	defer teardown()

	// Instantiate logger
	logger, err := New(&Config{
		Service:  "MyService",
		Instance: "MyInstance",
		Folder:   tempdir,
		Filename: "myservice",
		Rotation: ROT_DAILY,
		Out:      OUT_FILE_AND_STDOUT,
		Headers:  true,
		JSON:     false,
		Compress: true,
		Columns:  []int64{}, // Use default columns
	})

	if err != nil {
		fmt.Printf("Could not start logger: %s", err.Error())
		os.Exit(1)
	}

	// Log messages
	notify := logger.NewCaller("Example 1")

	notify(0, "Hello, World!")
}
