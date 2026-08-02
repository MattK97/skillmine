---
name: skillmine
description: Search the user's own Claude Code history to check whether a request has come up before, and if it repeats, draft a skill from what was actually done. Use when the user asks "should I make a skill for this", "have I asked this before", "what do I keep repeating", "search my history", or the equivalent in another language. Also use proactively when the user says a task should be done "as always", "same as last time", or "like we did before".
---

# skillmine

Finds repeated requests in the user's own conversation history and turns the
repetition into a skill draft. Everything runs locally; nothing leaves the machine.

## Step 1 — make sure the binary exists

```bash
skillmine -query test -top 1
```

If the command is not found, install it and say so:

```bash
go install github.com/MattK97/skillmine@latest
```

If Go is unavailable, stop and tell the user. Do not reimplement the search.

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
| `sessions` | how many distinct sessions they came from |
| `day_span` | days between the first and last match |
| `repeated` | true when hits >= 3 and sessions >= 2 |
| `verdict` | one-line plain-language conclusion |

## Step 4 — judge, and be willing to say no

`repeated: false` → say plainly that this looks like a one-off, and stop.
Do not manufacture a skill out of two coincidental matches.

`repeated: true` → **still read the matched prompt texts before continuing.**
Lexical matching can group prompts that merely share vocabulary. If the matches
are not the same intent, say so and stop.

Many matches inside a single session on a single day usually mean one debugging
run, not a habit. `sessions` and `day_span` are the signal, not `hits`.

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
- `-min-score` defaults to 0.10. Lower it to widen a search that returns too little.
- Matching cannot bridge synonyms that share no words. If recall looks poor, retry
  with a query mixing both vocabularies the user switches between.
