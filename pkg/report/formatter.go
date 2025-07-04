package report

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/wycleffsean/nostos/lang"
	"go.lsp.dev/uri"
)

// Formatter converts a NostosError into a formatted string.
type Formatter interface {
	Format(err error) string
}

// PrettyFormatter formats errors with color and source context.
type PrettyFormatter struct{}

// SimpleFormatter formats errors in a basic one line style.
type SimpleFormatter struct{}

func NewPrettyFormatter() Formatter { return &PrettyFormatter{} }
func NewSimpleFormatter() Formatter { return &SimpleFormatter{} }

func (f *SimpleFormatter) Format(err error) string {
	if ne, ok := err.(lang.NostosError); ok {
		pos := ne.Pos()
		path := uriPath(ne.URI())
		return fmt.Sprintf("%s:%d:%d: %s\n", path, pos.LineNumber, pos.CharacterOffset, err.Error())
	}
	return err.Error() + "\n"
}

func (f *PrettyFormatter) Format(err error) string {
	if ne, ok := err.(lang.NostosError); ok {
		return formatPretty(ne, err.Error())
	}
	return err.Error() + "\n"
}

func formatPretty(ne lang.NostosError, msg string) string {
	pos := ne.Pos()
	path := uriPath(ne.URI())
	header := color.New(color.Bold).Sprintf("%s:%d:%d", path, pos.LineNumber, pos.CharacterOffset)

	var sb strings.Builder
	sb.WriteString(header + "\n")
	sb.WriteString(color.New(color.FgRed).Sprint(msg) + "\n")

	// attempt to print source context
	if path != "" {
		if lines, err := readLines(path); err == nil {
			line := int(pos.LineNumber)
			start := line - 1
			if start < 1 {
				start = 1
			}
			end := line + 1
			if end > len(lines) {
				end = len(lines)
			}
			numWidth := len(fmt.Sprintf("%d", end))
			for i := start; i <= end; i++ {
				prefix := fmt.Sprintf("%*d | ", numWidth, i)
				sb.WriteString(prefix + lines[i-1] + "\n")
				if i == line {
					underline := strings.Repeat(" ", numWidth+3+int(pos.CharacterOffset))
					caretCount := 1
					if pos.ByteLength > 1 {
						caretCount = int(pos.ByteLength)
					}
					underline += color.New(color.FgRed).Sprint(strings.Repeat("^", caretCount))
					sb.WriteString(underline + "\n")
				}
			}
		}
	}
	return sb.String()
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func uriPath(u uri.URI) string {
	if u == "" {
		return ""
	}
	return u.Filename()
}
