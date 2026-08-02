---
name: skillmine
description: Search the user's own Claude Code history to check whether a request has come up before, and if it repeats, draft a skill from what was actually done. Use when the user asks "should I make a skill for this", "have I asked this before", "what do I keep repeating", "search my history", or the equivalent in another language. Also use proactively when the user says a task should be done "as always", "same as last time", or "like we did before".
---

# skillmine

Finds repeated requests in the user's own conversation history and turns the
repetition into a skill draft. Everything runs locally; nothing leaves the machine.

## Step 1 — locate the binary

Work down this list and stop at the first that succeeds. Remember which form
worked and reuse that exact command in every later step.

**1. Already on PATH:**

```bash
command -v skillmine
```

**2. Installed but not on PATH.** This is the common case, not an edge case:
`go install` writes to `$(go env GOPATH)/bin`, which many setups never export.

```bash
ls "$(go env GOPATH)/bin/skillmine"
```

**3. Not installed — build it from this plugin's own checkout.** The plugin
directory contains the full Go source, so nothing is downloaded.

Its path is the base directory given at the top of this skill with
`/skills/skillmine` removed. For example, a base directory of
`~/.claude/plugins/cache/skillmine/skillmine/0.1.0/skills/skillmine` makes the
plugin root `~/.claude/plugins/cache/skillmine/skillmine/0.1.0`.

Do not use `$CLAUDE_PLUGIN_ROOT`: it is set for hooks, not for shell commands,
and will be empty here.

```bash
mkdir -p "$(go env GOPATH)/bin"
cd "<plugin root>" && go build -o "$(go env GOPATH)/bin/skillmine" .
```

`cd` first. Passing the directory as an argument — `go build -o out <dir>` —
fails with "directory outside main module" whenever the working directory
belongs to a different module.

If this skill was not installed as a plugin, fetch it instead:

```bash
go install github.com/MattK97/skillmine@latest
```

**After building or installing, call it by full path** —
`"$(go env GOPATH)/bin/skillmine"` — not by name. PATH does not change inside a
running session, so plain `skillmine` will still fail and you will loop.

If `go` itself is missing, say so plainly, point the user at
<https://go.dev/dl/>, and stop. Do not fall back to grepping the transcripts by
hand: the ranking is the entire point of the tool, and a keyword grep produces
confident nonsense.

## Step 2 — decide the query

**If the user supplied a topic**, use it verbatim.

**If they did not**, read the current conversation and write the query yourself:
3–8 content words describing what this conversation is *about*. Use the words the
user actually typed rather than synonyms — matching is lexical, so vocabulary has
to line up.

Skip filler. `reset database products conversations keep account` is a good query;
`please clean up the database for me` is a bad one.

If the conversation covers several topics, pick the one being worked on now.

## Step 3 — search

```bash
skillmine -query "<query>" -format json -top 15
```

Read `summary`:

| field | meaning |
|---|---|
| `hits` | how many past prompts matched |
| `days` | how many distinct calendar days they fall on |
| `day_span` | days between the first and last match |
| `sessions` | informational only — a session id survives resumption and can span weeks |
| `recurring` | true when hits >= 3 across >= 3 distinct days |
| `broad_query` | true when the query had fewer than two content words |
| `guidance` | what you still have to decide |

## Step 4 — judge, and be willing to say no

**The tool does not decide whether something deserves a skill. You do.**

It can establish that a query matched on several separate days. It cannot tell
whether those matches are one request phrased differently or unrelated work that
happens to share a subject. That distinction is semantic, and measuring it from
the text was tried and does not work.

`broad_query: true` → the query was one word, so the results are a subject area.
Do not judge from this. Re-run with several words drawn from the actual request.

`recurring: false` → say plainly this looks like a one-off, and stop.

`recurring: true` → **read the matched prompt texts before going further.**
Ask yourself: is this the same request each time, or twelve different things
that all mention the same tool? A query like `supabase` will happily return
fifteen matches across nine days that share nothing but a product name.

If the matches are not one request, say so and stop. That is a useful answer,
not a failure.

## Step 5 — collect evidence

```bash
skillmine -query "<query>" -format json -top 8 -evidence
```

Each item holds three things:

| field | what it tells you |
|---|---|
| `prompt` | what the user asked for |
| `steps` | the tools actually invoked, in order — *how* it was done |
| `followups` | what the user said next, and what ran in response |

**`followups` is the most valuable field and the easiest to skip. Read it first.**

A correction there — "no, keep table X too", "you missed the storage bucket" —
is the best input available. It says what the naive first attempt got wrong, and
it belongs in the skill as an explicit step.

A confirmation — "works", "perfect" — means the procedure above it can be encoded
as-is.

No followups, or an immediate change of subject, means the run was uneventful.
Weight those lower than runs you can see were corrected.

## Step 6 — draft the skill

Write a `SKILL.md` draft:

- `name`: short, kebab-case
- `description`: what it does, plus the trigger phrases the user actually used —
  taken from the matched prompts, including their language and their typos
- body: the procedure generalised from `steps`. Parameterise what varied between
  runs (ids, table names, paths); keep what stayed the same.
- **fold every correction from `followups` in as an explicit step**, so the skill
  starts where the last run finished instead of repeating the same mistake
- call out destructive steps and require confirmation before them

Show the draft in chat. **Do not write it to disk unless the user says so.**
On approval, save to `~/.claude/skills/<name>/SKILL.md`.

## Notes

- Default corpus is `~/.claude/projects`. Override with `-dir`.
- The conversation you are in right now is excluded automatically, via
  `CLAUDE_CODE_SESSION_ID`. Without that, prompts from a few minutes ago count as
  history and everything under discussion looks like a long-standing habit. Pass
  `-exclude-session ""` to include it anyway.
- `-min-score` defaults to 0.10. Lower it to widen a search that returns too little.
- Matching cannot bridge synonyms that share no words. If recall looks poor, retry
  with a query mixing both vocabularies the user switches between.
