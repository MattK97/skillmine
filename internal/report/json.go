package report

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSONReporter renders a machine-readable report. The summary carries a
// ready-made verdict so the consumer does not have to recompute it.
type JSONReporter struct {
	Indent bool
}

// Report writes the result to w as JSON.
func (j JSONReporter) Report(w io.Writer, res Result) error {
	enc := json.NewEncoder(w)
	if j.Indent {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(res); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	return nil
}
