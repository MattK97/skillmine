// Package search ranks past prompts by similarity to a query, using TF-IDF
// with cosine similarity.
//
// TF-IDF is chosen over set-overlap measures such as Jaccard because prompts
// are conversational: the same request phrased twice shares only two or three
// meaningful words buried in a dozen filler words. Jaccard weighs every token
// equally and scores those pairs near zero. IDF suppresses the filler.
package search

import (
	"math"
	"sort"

	"github.com/MattK97/skillmine/internal/transcript"
)

// Hit is one search result.
type Hit struct {
	Record transcript.Record `json:"record"`
	Score  float64           `json:"score"`
	Index  int               `json:"-"` // position in the corpus
}

// Index is a prepared corpus, ready to be queried.
type Index struct {
	docs []transcript.Record
	idf  map[string]float64
	vecs []map[string]float64
}

// New builds an index over the given prompts.
func New(docs []transcript.Record) *Index {
	x := &Index{
		docs: docs,
		idf:  make(map[string]float64),
		vecs: make([]map[string]float64, len(docs)),
	}

	df := make(map[string]int)
	toks := make([][]string, len(docs))
	for i, d := range docs {
		toks[i] = tokenize(d.Text)
		for w := range uniq(toks[i]) {
			df[w]++
		}
	}

	n := float64(len(docs))
	for w, c := range df {
		x.idf[w] = math.Log(n / float64(1+c))
	}

	for i := range docs {
		x.vecs[i] = x.vector(toks[i])
	}
	return x
}

// Len reports the number of indexed prompts.
func (x *Index) Len() int { return len(x.docs) }

// Doc returns the prompt at index i.
func (x *Index) Doc(i int) transcript.Record { return x.docs[i] }

// Search returns up to top hits scoring at least minScore, best first.
// A top of zero or less returns every hit above the threshold.
func (x *Index) Search(query string, top int, minScore float64) []Hit {
	q := x.vector(tokenize(query))
	if len(q) == 0 {
		return nil
	}

	var hits []Hit
	for i, v := range x.vecs {
		s := dot(q, v)
		if s < minScore {
			continue
		}
		hits = append(hits, Hit{Record: x.docs[i], Score: s, Index: i})
	}

	sort.Slice(hits, func(a, b int) bool {
		if hits[a].Score != hits[b].Score {
			return hits[a].Score > hits[b].Score
		}
		// Break ties by recency so results are stable between runs.
		return hits[a].Record.At.After(hits[b].Record.At)
	})

	if top > 0 && len(hits) > top {
		hits = hits[:top]
	}
	return hits
}

// vector builds a length-normalised TF-IDF vector.
func (x *Index) vector(toks []string) map[string]float64 {
	if len(toks) == 0 {
		return nil
	}

	tf := make(map[string]int, len(toks))
	for _, w := range toks {
		tf[w]++
	}

	v := make(map[string]float64, len(tf))
	var sum float64
	for w, n := range tf {
		idf, ok := x.idf[w]
		if !ok || idf <= 0 {
			// Unknown to the corpus, or present in nearly every document.
			continue
		}
		weight := (1 + math.Log(float64(n))) * idf
		v[w] = weight
		sum += weight * weight
	}
	if sum == 0 {
		return nil
	}

	norm := math.Sqrt(sum)
	for w := range v {
		v[w] /= norm
	}
	return v
}

// dot is the inner product of two sparse vectors. Both are length-normalised,
// so the inner product is the cosine similarity.
func dot(a, b map[string]float64) float64 {
	small, large := a, b
	if len(b) < len(a) {
		small, large = b, a
	}
	var s float64
	for w, x := range small {
		s += x * large[w]
	}
	return s
}

func uniq(toks []string) map[string]struct{} {
	set := make(map[string]struct{}, len(toks))
	for _, w := range toks {
		set[w] = struct{}{}
	}
	return set
}
