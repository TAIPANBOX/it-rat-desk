# it-rat-desk invariants

`@claude` 2026-11-01: a local prototype of a desk assistant for it-rat.com's
"Before you go" chat. Each invariant names what holds it, with the weakest
marker that is true: `(test: ...)`, `(gate: ...)`, `(partly gated: ...)` or
`(not enforced)`.

`@decided 2026-11-01`: the prototype is kept working and is not wired into the
live site. No relay runs on the internet; the live site keeps its deterministic
desk agent. The reason given: a public assistant can be used for anything, and
the site has no traffic yet to justify one.

`@decided 2026-11-01`: Claude Haiku 5.5 is the model. The OpenRouter free routes
stay in the code, off unless `OPENROUTER_DESK_KEY` is set: on the day they were
measured they answered in 15 to 54 seconds, and with the opt-out from training
on prompts only 2 of 13 free tool-capable models answered at all.

`@decided 2026-11-01`: the chat carries no buttons; writing to the team lives on
the team's address beside the chat, as a note form opened by pointing at it
(tapping on a phone). The answer streams and is shown from its first line.
Voice input is the browser's own speech recognition.

## Invariants

1. **The API key never leaves the server.** The browser talks only to the
   local server; nothing in `widget/` names a key or a provider host.
   *(gate: the estate secret scan in `.github/workflows/secretscan.yml`; that
   the widget carries no provider host is not enforced)*
2. **The server answers only on this machine.** It holds a key and answers
   anyone who can connect, so it refuses to listen anywhere but loopback.
   *(test: `TestLoopbackOnly`)*
3. **The model reads only the `.md` pages the site's `llms.txt` lists and that
   exist on disk.** Not `CLAUDE.md`, not `SEO-STATE.md`, not `llms-full.txt`,
   not a path with `..`. *(test: `TestAllowlistIsExactlyTheListedPagesOnDisk`,
   `TestReadPageRefusesEverythingElse`, `FuzzReadPage`, in CI for 30 s a run)*
4. **A conversation is checked before the model sees it**: alternating turns
   from the visitor, at most 20, each at most 2000 characters, ending on the
   visitor; anything else is a plain 400 and nothing streams.
   *(test: `TestValidateHistory`, `TestAskRefusesBadRequestsBeforeStreaming`)*
5. **Narration before reading is not part of the answer; an answer written
   before offering the team is.** *(test: `TestKeepTurnText`,
   `TestAskStreamsInOrderAndEndsWithDone`)*
6. **A stream always ends with one `done` line that carries the final answer**,
   which the widget draws over whatever streamed. *(test:
   `TestAskStreamsInOrderAndEndsWithDone`)*
7. **Every scenario in `features/` names one test that exists, and every test
   is named by a scenario** unless `unbound` in `scenarios_test.go` says why.
   *(test: `TestScenariosAreBound`; teeth: `scripts/gates-have-teeth.sh`, 7
   cases, counted from its own "OK: 7 cases" on 2026-11-01)*

## Gates

```sh
gofmt -l . && go vet ./... && go test -race ./...
go test -run '^$' -fuzz '^FuzzReadPage$' -fuzztime 30s .
scripts/gates-have-teeth.sh
```

## Conventions

- Public repository: no quotes of a person and no attributions of a decision;
  decisions are paraphrased under `@decided`.
- No long dashes in prose a reader sees (README, this file).
- Nothing metered is enabled without telling the owner first. The relay on the
  internet that the README sketches is exactly such a change.
