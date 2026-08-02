// Command skillmine searches your own Claude Code conversation history to find
// out whether a request has come up before, and shows what was actually done
// each time.
//
//	skillmine -query "reset the database"
//	skillmine -query "reset the database" -evidence
//	skillmine -query "reset the database" -format json
//
// Nothing leaves the machine.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/MattK97/skillmine/internal/evidence"
	"github.com/MattK97/skillmine/internal/report"
	"github.com/MattK97/skillmine/internal/search"
	"github.com/MattK97/skillmine/internal/transcript"
)

// Reporter is declared here, in the consumer, rather than in package report.
// Both reporters satisfy it structurally and neither imports this package.
type Reporter interface {
	Report(w io.Writer, res report.Result) error
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// run holds the logic and returns errors rather than exiting, because os.Exit
// skips deferred calls.
func run() error {
	query := flag.String("query", "", "what to look for in the history (required)")
	dir := flag.String("dir", defaultDir(), "directory holding the transcripts")
	top := flag.Int("top", 15, "how many hits to show")
	minScore := flag.Float64("min-score", 0.10, "similarity threshold, 0..1")
	format := flag.String("format", "text", "output format: text or json")
	withEvidence := flag.Bool("evidence", false, "include what was done after each hit")
	flag.Parse()

	if *query == "" {
		return errors.New("missing required flag -query")
	}
	// The flag package consumes the next argument as a value even when it looks
	// like a flag, so "-query -format json" silently sets query to "-format".
	// These two checks turn that into a clear error.
	if strings.HasPrefix(*query, "-") {
		return fmt.Errorf("-query looks like a flag, not a query: %q", *query)
	}
	if flag.NArg() > 0 {
		return fmt.Errorf("unexpected arguments: %v (quote queries containing spaces)", flag.Args())
	}

	reporter, err := newReporter(*format)
	if err != nil {
		return err
	}

	recs, err := transcript.Load(*dir)
	if err != nil {
		return err
	}

	idx := search.New(transcript.Prompts(recs))

	// Summarise every hit above the threshold, then truncate for display.
	// Otherwise -top would change the verdict.
	all := idx.Search(*query, 0, *minScore)
	summary := report.Summarize(all)

	shown := all
	if *top > 0 && len(shown) > *top {
		shown = shown[:*top]
	}

	res := report.Result{Query: *query, Summary: summary, Hits: shown}
	if *withEvidence {
		res.Evidence = evidence.Collect(recs, uuidsOf(shown))
	}

	return reporter.Report(os.Stdout, res)
}

func newReporter(format string) (Reporter, error) {
	switch format {
	case "text":
		return report.TextReporter{}, nil
	case "json":
		return report.JSONReporter{Indent: true}, nil
	default:
		return nil, fmt.Errorf("unknown format %q (allowed: text, json)", format)
	}
}

func uuidsOf(hits []search.Hit) []string {
	out := make([]string, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.Record.UUID)
	}
	return out
}

func defaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".claude", "projects")
	}
	return filepath.Join(home, ".claude", "projects")
}
