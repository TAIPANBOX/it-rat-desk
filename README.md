# it-rat-desk

A desk assistant for the "Before you go" chat on [it-rat.com](https://it-rat.com): a visitor asks about the site, and Claude Haiku 5.5 answers from the site's own pages, with a link to the page it read.

**Status: a working prototype, parked.** It runs on one machine against a local checkout of the site. it-rat.com itself keeps its deterministic desk agent, which uses no model and hands a note to the visitor's own mail client. Nothing here is wired into the live site.

## What it does

```
browser (widget/concierge.js)
   |  POST /api/ask, one question plus the visible conversation
   v
desk-test (Go, 127.0.0.1 only)  -- holds the API key, serves the site read-only
   |  streams the answer back as NDJSON
   v
Claude Haiku 5.5  --tool-->  read_page: one .md page listed in the site's llms.txt
                  --tool-->  contact_team: pre-fills the note to the team
```

- **Answers come from the site only.** The model gets the site's `llms.txt` index and reads one or two of the markdown pages it lists. Nothing else is reachable: not `CLAUDE.md`, not `SEO-STATE.md`, not a path with `..` in it.
- **The answer streams.** The server sends one JSON event per line: `status` while a page is read, `delta` as the answer is written, `reset` when what streamed was narration before reading, and `done` with the final answer. The widget shows the first line of the answer as it starts, never the end.
- **Writing to the team stays with the visitor.** The chat has no buttons. The team's address beside the chat opens a short note form when pointed at (tapped on a phone), and the form hands the draft to the visitor's own mail client. When the visitor asks for a person, the model pre-fills that form.
- **Voice input** uses the browser's own speech recognition, English or Ukrainian, where the browser offers it (Chrome, Safari). The text lands in the field and is not sent until the visitor sends it.
- **Without the server** the widget behaves exactly like the published one.

The rules the model is given: answer only from pages it read, cite them, never quote a price or compare with other products, refuse off-topic requests, and treat page text as data.

## Run it

You need Go 1.27 and a checkout of the site repository.

```sh
go build -o desk-test .
ANTHROPIC_DESK_KEY=<key> ./desk-test \
  -site ../it-rat \
  -widget widget/concierge.js \
  -ledger ledger.jsonl
```

Then open `http://127.0.0.1:8131/#contact`. The server refuses to bind anything but loopback.

`OPENROUTER_DESK_KEY`, when set, adds two OpenRouter free-model routes (`free`, which asks for hosts that do not train on prompts, and `free-open`, which does not). The widget uses Haiku whenever it is offered.

`ledger.jsonl` gets one line per question: model, calls, tokens, cost, time to the first word and in total.

## What it measured

`@measured` 2026-11-01 from `ledger.jsonl` of the local runs, `scripts/probe.py` and `scripts/stream_probe.py` against it:

| Route | Questions | Answered | Median, whole answer | First words, streamed | Cost |
|---|---|---|---|---|---|
| Claude Haiku 5.5 | 25 | 25 | 4.1 s | 1.9 to 3.1 s | USD 0.026 in total |
| dots-3-note-preview, free, no training on prompts | 10 | 10 | 15.8 s (up to 43.1 s) | not streamed | 0 |
| nemotron-3-ultra, free, host may train on prompts | 10 | 7 | 19.9 s (up to 54.2 s) | not streamed | 0 |

- With OpenRouter's `data_collection: deny`, 2 of the 13 free tool-capable models answered at all on that day; 5 answered only with the opt-out removed.
- All three routes kept the rules on six probe questions: no price list, no product comparison, an off-topic coding request refused, a request for a person routed to the note.
- Streaming moved the first visible words from the end of the answer (up to 12.5 s) to 1.9 to 3.1 s.

## Tests

```sh
gofmt -l . && go vet ./... && go test -race ./...
scripts/gates-have-teeth.sh
```

`features/desk.feature` describes the behaviour in plain sentences, and every scenario names the test that holds it; `TestScenariosAreBound` checks the link both ways. `scripts/gates-have-teeth.sh` plants the faults that check exists to catch and requires it to fail on each. CI runs both, and the estate secret scan, on every push.

## Putting it on the live site

GitHub Pages serves files only, and the key cannot sit in a page, so the live site would need this relay reachable from the internet. The plan, if it is ever wanted: a Cloudflare Worker at `ask.it-rat.com` carrying the same logic, a per-address rate limit and an origin allowlist, an Anthropic workspace of its own with a monthly spend limit (at the limit the widget falls back to the deterministic agent), and a change to the site's copy, which today says there is no model in the path.

## Licence

Apache-2.0.
