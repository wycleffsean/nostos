package report

import (
	"fmt"
	"io"
)

// Reporter prints formatted errors.
type Reporter struct {
	Formatter Formatter
	Out       io.Writer
}

// New creates a Reporter using the provided formatter and writer.
func New(f Formatter, out io.Writer) *Reporter {
	return &Reporter{Formatter: f, Out: out}
}

// Report prints the provided errors. If multiple errors are provided, a summary
// count is printed at the end.
func (r *Reporter) Report(errs []error) {
	if len(errs) == 0 {
		return
	}
	for _, err := range errs {
		fmt.Fprint(r.Out, r.Formatter.Format(err))
	}
	if len(errs) > 1 {
		fmt.Fprintf(r.Out, "\n%d errors\n", len(errs))
	}
}
