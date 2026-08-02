# skillmine

Search your own Claude Code history to find out whether you have asked for
something before — and see what was actually done each time.

No dependencies. No network calls. Everything stays on your machine.

## Why

You ask for the same thing over and over without noticing, because you phrase it
differently every time. "reset the test database", "wipe the db again",
"do the usual cleanup before the demo" — three ways of saying one thing, spread
over two months.

`skillmine` finds those, and more importantly shows **what the assistant actually
ran** in response, plus **how you reacted**. That is the material a reusable skill
is made of. A skill written from your prompts alone just restates the request.

## Install

```bash
go install github.com/MattK97/skillmine@latest
```

Optionally install the bundled skill so your agent can run the tool itself:

```bash
git clone https://github.com/MattK97/skillmine
cp -r skillmine/skills/skillmine ~/.claude/skills/
```

## Usage

```bash
skillmine -query "reset the database"
```

```
Query: "reset the database"

 1. 0.460  2026-04-01  demo-app     do the usual database reset before I record the demo video
 2. 0.418  2026-03-02  demo-app     reset the test database, keep only my user account and phone number
 3. 0.219  2026-04-20  demo-app     clean the database, products and the message history need to go

4 hits · 3 sessions · 1 project · 50 days (2026-03-02 .. 2026-04-20)
Verdict: repeated — worth turning into a skill
```

Add `-evidence` to see what happened next:

```
[1] 2026-04-01  ASKED: database cleanup: drop products, vehicles and conversations…
    -> Bash    {"command": "psql $DATABASE_URL -c 'select count(*) from products'"}
    -> Bash    {"command": "psql $DATABASE_URL -c 'begin; delete from products; …

    USER REPLIED: no wait, you also need to purge the storage bucket
    -> Bash    {"command": "psql $DATABASE_URL -c 'delete from storage_objects …

    USER REPLIED: perfect, that is it
    (no tool calls)
```

That `USER REPLIED` line is the point. The correction says what the first attempt
missed, so the skill you write from it starts where the last run finished.

For agents, `-format json` emits the same data with a precomputed verdict.

## Flags

| flag | default | meaning |
|---|---|---|
| `-query` | — | what to look for (required) |
| `-dir` | `~/.claude/projects` | where transcripts live |
| `-top` | 15 | how many hits to show |
| `-min-score` | 0.10 | similarity threshold, 0..1 |
| `-format` | `text` | `text` or `json` |
| `-evidence` | off | include tool calls and your replies |

Truncating the display with `-top` never changes the verdict — the summary is
always computed over every hit above the threshold.

## Using it from an agent

`skills/skillmine/SKILL.md` teaches an agent to run the tool on its own: derive a
query from the current conversation, check whether the topic repeats, pull the
evidence, and draft a skill from it. It works with anything that can run a shell
command — Claude Code, and other agent CLIs alike. No MCP server needed.

## How it works

TF-IDF with cosine similarity over the user prompts in
`~/.claude/projects/**/*.jsonl`.

The method was chosen by measurement. On a set of seven real prompts asking for
the same thing in different words:

| method | result |
|---|---|
| Jaccard over word trigrams | 0 of 21 pairs linked |
| Jaccard over character 4-grams | 2 of 21 |
| Jaccard over 5-character stems | 3 of 21 |
| **TF-IDF + cosine** | **6 of 7 targets retrieved, no false positives** |

Jaccard fails because it weighs filler words like "okay" and "actually" the same
as "cleanup" and "database". Conversational prompts are mostly filler, so the
signal drowns. IDF suppresses terms that appear everywhere.

Text is lowercased, stripped of diacritics, split on non-alphanumerics, filtered
against a stop-word list, and truncated to 6-character stems. That last step is a
crude stemmer, but it collapses inflected forms across languages without pulling
in a dependency.

### The golden test

`testdata/sample.jsonl` holds seven paraphrases of one request among 32
distractors, several of which deliberately share database vocabulary.
`internal/search/tfidf_test.go` asserts two properties: enough of the seven are
retrieved, and the top of the ranking contains no distractors.

The test asserts behaviour, not implementation. Replace the whole similarity
method and it will still tell you whether the replacement is better. It also
pins the threshold, so `-min-score` is measured rather than guessed.

## Limitations

- **Matching is lexical.** Two phrasings that share no words will not be linked,
  even when they mean the same thing. Mixing vocabularies in the query helps.
- **No discovery mode.** You need a topic, or an agent that derives one from the
  conversation. Clustering the whole corpus is not implemented.
- **File order is treated as chronological.** Evidence collection assumes
  `.jsonl` files are appended sequentially, which may not hold for branched
  sessions.

## Layout

```
main.go                      flags, Reporter interface, wiring
internal/transcript/         .jsonl reading, polymorphic content decoding
internal/search/             TF-IDF, cosine similarity, the golden test
internal/evidence/           what ran after a prompt, and how the user replied
internal/report/             text and JSON output
skills/skillmine/SKILL.md    instructions for an agent
testdata/sample.jsonl        synthetic fixture, no real conversation data
```

## Development

```bash
go test ./... -race
go vet ./...
```

The fixture is synthetic on purpose. Transcripts contain private conversations,
absolute paths and project identifiers, so none are committed here.

## License

MIT
