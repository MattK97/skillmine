package transcript

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// maxLine raises the Scanner line limit. Transcript lines routinely exceed the
// 64 KB default (pasted logs, file contents, base64 images), and Scanner stops
// reading silently when a line does not fit.
const maxLine = 16 << 20

// ScanFile reads one transcript stream.
func ScanFile(src io.Reader, project string) ([]Record, error) {
	var out []Record

	sc := bufio.NewScanner(src)
	sc.Buffer(make([]byte, 0, 64*1024), maxLine)

	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}

		var raw rawRecord
		if err := json.Unmarshal(line, &raw); err != nil {
			// Transcripts can be truncated mid-write when a session dies.
			// One bad line must not fail the whole file.
			continue
		}
		if rec, ok := raw.toRecord(project); ok {
			out = append(out, rec)
		}
	}

	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("reading transcript: %w", err)
	}
	return out, nil
}

// Load reads every .jsonl transcript under root, skipping duplicate UUIDs.
//
// Record order within a file is preserved; package evidence relies on it to
// walk forward from a prompt through the responses that followed.
func Load(root string) ([]Record, error) {
	var all []Record
	seen := make(map[string]bool)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}

		recs, err := loadOne(path)
		if err != nil {
			return err
		}
		for _, r := range recs {
			if seen[r.UUID] {
				continue
			}
			seen[r.UUID] = true
			all = append(all, r)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", root, err)
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("%s: no transcripts found", root)
	}
	return all, nil
}

// loadOne exists as a separate function so the deferred Close runs after each
// file rather than after the entire walk.
func loadOne(path string) ([]Record, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	recs, err := ScanFile(f, projectName(filepath.Dir(path)))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return recs, nil
}

// projectName turns an encoded transcript directory into a readable label:
//
//	-Users-alice-work-api  ->  work-api
//
// The encoding is lossy (underscores also became dashes), so this aims for
// recognisable rather than exact.
func projectName(dir string) string {
	base := strings.TrimPrefix(filepath.Base(dir), "-")
	parts := strings.Split(base, "-")
	if len(parts) > 2 && strings.EqualFold(parts[0], "Users") {
		return strings.Join(parts[2:], "-")
	}
	return base
}
