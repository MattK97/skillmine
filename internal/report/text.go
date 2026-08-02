package report

import (
	"fmt"
	"io"

	"github.com/MattK97/skillmine/internal/transcript"
)

// TextReporter renders a human-readable report.
type TextReporter struct{}

// Report writes the result to w.
func (TextReporter) Report(w io.Writer, res Result) error {
	if _, err := fmt.Fprintf(w, "Query: %q\n\n", res.Query); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}

	if len(res.Hits) == 0 {
		_, err := fmt.Fprintln(w, "No matches. This topic is not in your history.")
		return err
	}

	for i, h := range res.Hits {
		fmt.Fprintf(w, "%2d. %.3f  %s  %-20s %s\n",
			i+1,
			h.Score,
			h.Record.At.Format("2006-01-02"),
			shorten(h.Record.Project, 20),
			shorten(h.Record.Text, 90),
		)
	}

	s := res.Summary
	fmt.Fprintf(w, "\n%s (%s .. %s), %d %s\n",
		s.Spread,
		s.First.Format("2006-01-02"), s.Last.Format("2006-01-02"),
		s.Projects, plural(s.Projects, "project", "projects"))
	fmt.Fprintf(w, "%s\n", s.Guidance)

	if len(res.Evidence) > 0 {
		return writeEvidence(w, res)
	}
	return nil
}

func writeEvidence(w io.Writer, res Result) error {
	if _, err := fmt.Fprint(w, "\n=== WHAT WAS ACTUALLY DONE ===\n"); err != nil {
		return fmt.Errorf("writing evidence: %w", err)
	}

	for i, item := range res.Evidence {
		fmt.Fprintf(w, "\n[%d] %s  ASKED: %s\n",
			i+1,
			item.Prompt.At.Format("2006-01-02"),
			shorten(item.Prompt.Text, 100))

		writeSteps(w, item.Steps)

		// A correction here says what the first attempt missed.
		for _, f := range item.Followups {
			fmt.Fprintf(w, "\n    USER REPLIED: %s\n", shorten(f.Text, 200))
			writeSteps(w, f.Steps)
		}
	}
	return nil
}

func writeSteps(w io.Writer, steps []transcript.Turn) {
	if len(steps) == 0 {
		fmt.Fprintln(w, "    (no tool calls)")
		return
	}
	for _, s := range steps {
		fmt.Fprintf(w, "    -> %-28s %s\n", s.Tool, shorten(s.Args, 100))
	}
}
