package server

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fatih/color"
	"golang.org/x/crypto/ssh/terminal"
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

	plogsStr, pbytesStr := prettyParsedSums(plogs, pbytes)

	return plogsStr, pbytesStr, plogs, pbytes

}

// prettyParsedSums turns int64 into pretty strings
func prettyParsedSums(plogs, pbytes int64) (plogsStr, pbytesStr string) {

	// Add thousands separator for parsed logs
	plogsStrNorm := strconv.FormatInt(plogs, 10)
	seps := int(len(plogsStrNorm)/3) + 1
	plogsTsd := make([]string, seps)
	for i := 1; i <= seps; i++ {
		if len(plogsStrNorm)-i*3 >= 0 {
			plogsTsd[seps-i] = plogsStrNorm[len(plogsStrNorm)-i*3 : len(plogsStrNorm)-(i-1)*3]
		} else {
			plogsTsd[seps-i] = plogsStrNorm[:len(plogsStrNorm)-(i-1)*3]
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
		strings.TrimSpace(fmt.Sprintf("%6.2f %s", pbytesNorm, pbytesSuffix))
}

// floatSorter implements the sort.Interface
type floatSorter struct {
	order  []int
	floats []float64
}

// Len implements sort.Interface.Len
func (f *floatSorter) Len() int {
	if f.order == nil {
		f.createOrders()
	}

	return len(f.floats)
}

// Less implements sort.Interface.Less
func (f *floatSorter) Less(i, j int) bool {
	return f.floats[i] < f.floats[j]
}

// Swap implements sort.Interface.Swap
func (f *floatSorter) Swap(i, j int) {

	tempIdx := f.order[i]
	tempV := f.floats[i]

	f.floats[i] = f.floats[j]
	f.floats[j] = tempV

	f.order[i] = f.order[j]
	f.order[j] = tempIdx

}

// createOrders initiates the order slice
func (f *floatSorter) createOrders() {
	f.order = make([]int, len(f.floats))
	for i := 0; i < len(f.floats); i++ {
		f.order[i] = i
	}
}

// GetSortedIndexes returns the indexes of sorted floats
func (f *floatSorter) GetIndexes() []int {
	if f.order == nil {
		f.createOrders()
	}
	return f.order
}

// getOffset returns the available tty space
func getOffset(width int) int {
	w, _, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0
	}
	offset := int((w - width) / 2)
	if offset < 0 {
		return 0
	}
	return offset
}

// centerStr centers a string
func centerStr(value string) string {
	width := utf8.RuneCountInString(value)
	offset := getOffset(width)

	return fmt.Sprintf("%s%s", strings.Repeat(" ", offset), value)
}

// barchart draws a rudimentary bar chart
func barchart(dst io.Writer, ticks []interface{}, values []float64, blockchar string, c *color.Color, maxHeight, sep int, center bool) {
	var usechar string

	// Precalculate some statistics
	realHeight := 0
	barwidth := 0
	lineWidth := 0
	for i := range values {
		tStr := fmt.Sprintf("%v", ticks[i])
		if width := len(tStr); width > barwidth {
			barwidth = width
		}

		v := values[i]
		if barHeight := int(float64(maxHeight) * float64(v)); barHeight > realHeight {
			realHeight = barHeight
		}
		lineWidth += utf8.RuneCountInString(tStr) + sep
	}
	lineWidth += 10 // ylabel+space+bar+space
	offset := getOffset(lineWidth)
	realHeight++

	for j := realHeight; j >= -1; j-- {
		line := bytes.NewBufferString("")

		for i, tick := range ticks {

			// X-Axis
			if j == 0 {
				if i == 0 {
					line.WriteString(fmt.Sprintf("%s%s", strings.Repeat(" ", 8), "┗━"))
				}
				line.WriteString(fmt.Sprintf("%s", strings.Repeat("━", barwidth+sep)))
				continue
			}

			// Ticks
			if j == -1 {
				if i == 0 {
					line.WriteString(strings.Repeat(" ", 10))
				}
				line.WriteString(fmt.Sprintf("%v%s", tick, strings.Repeat(" ", sep)))
				continue
			}

			// Bars
			if i == 0 {
				if realHeight < 5 || j%2 == realHeight%2 {
					share := fmt.Sprintf("%6.2f%%", float64(j)/float64(realHeight)*100)
					line.WriteString(fmt.Sprintf("%-7s %s ", share, "┃"))
				} else {
					line.WriteString(fmt.Sprintf("%-7s %s ", "", "┃"))
				}
			}
			barHeight := int(float64(maxHeight) * values[i])

			if barHeight >= j {
				usechar = c.Sprint(blockchar)
			} else {
				usechar = " "
			}
			line.WriteString(fmt.Sprintf("%s%s", strings.Repeat(usechar, barwidth), strings.Repeat(" ", sep)))

		}
		lineStr := line.String()
		if center {
			lineStr = fmt.Sprintf("%s%s", strings.Repeat(" ", offset), lineStr)
		}
		dst.Write([]byte(fmt.Sprintf("%s\n", lineStr)))
	}
}
