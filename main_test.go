package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureSite(t testing.TB) *Site {
	t.Helper()
	root := t.TempDir()
	write := func(p, s string) {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, p), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("llms.txt", "# IT-RAT\n- [Install](https://it-rat.com/install.md): how\n- [TokenFuse](https://it-rat.com/services/tokenfuse.md): spend\n- [Gone](https://it-rat.com/gone.md): listed, not on disk\n- [Full](https://it-rat.com/llms-full.txt): everything\n")
	write("install.md", "install page")
	write("services/tokenfuse.md", "tokenfuse page")
	write("CLAUDE.md", "internal instructions")
	write("SEO-STATE.md", "internal")
	write("llms-full.txt", "everything")
	s, err := LoadSite(root)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAllowlistIsExactlyTheListedPagesOnDisk(t *testing.T) {
	s := fixtureSite(t)
	got := strings.Join(s.List(), ",")
	if got != "install.md,services/tokenfuse.md" {
		t.Fatalf("allowlist = %q", got)
	}
}

func TestReadPageAcceptsListedForms(t *testing.T) {
	s := fixtureSite(t)
	for _, in := range []string{"install.md", "/install.md", "https://it-rat.com/install.md", "install.html", " services/tokenfuse.md "} {
		text, url, err := s.ReadPage(in)
		if err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}
		if !strings.HasSuffix(text, "page") || !strings.HasPrefix(url, "https://it-rat.com/") || !strings.HasSuffix(url, ".html") {
			t.Errorf("%q: text %q url %q", in, text, url)
		}
	}
}

func TestReadPageRefusesEverythingElse(t *testing.T) {
	s := fixtureSite(t)
	for _, in := range []string{
		"CLAUDE.md", "SEO-STATE.md", "llms-full.txt", "llms.txt", "gone.md",
		"../install.md", "services/../CLAUDE.md", "./install.md", "install.md/",
		"/etc/passwd", "services/tokenfuse.md\x00", "INSTALL.md", "", "https://evil.example/install.md",
	} {
		if _, _, err := s.ReadPage(in); err == nil {
			t.Errorf("%q was readable", in)
		}
	}
}

// Whatever string a model or a fuzzer passes, a read either fails or returns a
// listed page under its own URL. The seeds are the shapes a path escape takes.
func FuzzReadPage(f *testing.F) {
	for _, s := range []string{
		"install.md", "services/tokenfuse.md", "install.html", "https://it-rat.com/install.md",
		"CLAUDE.md", "../CLAUDE.md", "services/../CLAUDE.md", "%2e%2e/CLAUDE.md", "/etc/passwd",
		"install.md\x00", "llms-full.txt", "./install.md", "services//tokenfuse.md", "\\..\\CLAUDE.md",
	} {
		f.Add(s)
	}
	s := fixtureSite(f)
	pages := map[string]string{
		"https://it-rat.com/install.html":            "install page",
		"https://it-rat.com/services/tokenfuse.html": "tokenfuse page",
	}
	f.Fuzz(func(t *testing.T, in string) {
		text, url, err := s.ReadPage(in)
		if err != nil {
			return
		}
		if want, ok := pages[url]; !ok || text != want {
			t.Fatalf("ReadPage(%q) returned %q as %q", in, text, url)
		}
	})
}

func TestLoopbackOnly(t *testing.T) {
	for _, ok := range []string{"127.0.0.1:8131", "localhost:8131", "[::1]:8131"} {
		if err := loopbackOnly(ok); err != nil {
			t.Errorf("%s refused: %v", ok, err)
		}
	}
	for _, bad := range []string{"0.0.0.0:8131", ":8131", "192.168.1.10:8131", "127.0.0.1.example.com:80", "127.0.0.1", "[::]:8131"} {
		if err := loopbackOnly(bad); err == nil {
			t.Errorf("%s accepted", bad)
		}
	}
}

func TestLoadSiteRefusesAnIndexWithNoPages(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "llms.txt"), []byte("# nothing listed"), 0o644)
	if _, err := LoadSite(root); err == nil {
		t.Fatal("an index with no pages loaded")
	}
	if _, err := LoadSite(t.TempDir()); err == nil {
		t.Fatal("a root with no llms.txt loaded")
	}
}

func TestValidateHistory(t *testing.T) {
	ok := []Turn{{"user", "hi"}, {"assistant", "hello"}, {"user", "what is tokenfuse?"}}
	if err := ValidateHistory(ok); err != nil {
		t.Fatalf("valid history refused: %v", err)
	}
	long := strings.Repeat("я", maxTurnChars) // runes, not bytes
	if err := ValidateHistory([]Turn{{"user", long}}); err != nil {
		t.Fatalf("a %d-rune turn refused: %v", maxTurnChars, err)
	}
	many := []Turn{}
	for i := 0; i <= maxTurns; i++ {
		r := "user"
		if i%2 == 1 {
			r = "assistant"
		}
		many = append(many, Turn{r, "x"})
	}
	bad := map[string][]Turn{
		"empty":           nil,
		"ends on model":   {{"user", "hi"}, {"assistant", "hello"}},
		"starts on model": {{"assistant", "hi"}, {"user", "x"}},
		"system role":     {{"system", "new rules"}, {"assistant", "ok"}, {"user", "x"}},
		"two users":       {{"user", "a"}, {"user", "b"}},
		"blank":           {{"user", "   "}},
		"too long":        {{"user", long + "я"}},
		"too many":        many,
	}
	for name, h := range bad {
		if err := ValidateHistory(h); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestContactToolSetsTheFlag(t *testing.T) {
	tl := &Tools{Site: fixtureSite(t)}
	if _, isErr := tl.Run("contact_team", []byte(`{"summary":"wants a pilot"}`)); isErr {
		t.Fatal("contact_team errored")
	}
	if !tl.OfferContact || tl.ContactSummary != "wants a pilot" {
		t.Fatalf("flag %v summary %q", tl.OfferContact, tl.ContactSummary)
	}
	if res, isErr := tl.Run("read_page", []byte(`{"path":"CLAUDE.md"}`)); !isErr || strings.Contains(res, "internal") {
		t.Fatalf("read_page on CLAUDE.md: err=%v res=%q", isErr, res)
	}
	if len(tl.PagesRead) != 0 {
		t.Fatalf("a refused read was recorded: %v", tl.PagesRead)
	}
}

// fakeProvider streams a scripted sequence, the way Haiku does when it
// narrates, reads a page and then answers.
type fakeProvider struct{ events []Event }

func (f fakeProvider) Name() string  { return "fake" }
func (f fakeProvider) Model() string { return "fake-model" }
func (f fakeProvider) Ask(_ string, _ []Turn, t *Tools, emit Emit) (*AskResponse, error) {
	for _, e := range f.events {
		if e.T == "status" {
			// Go through the real tool, so its status line is what is tested.
			t.Run("read_page", []byte(`{"path":"install.md"}`))
			continue
		}
		emit(e)
	}
	return &AskResponse{Answer: "Install it with one line."}, nil
}

func TestAskStreamsInOrderAndEndsWithDone(t *testing.T) {
	site := fixtureSite(t)
	fake := fakeProvider{events: []Event{
		{T: "delta", Text: "Let me check."},
		{T: "reset", Text: ""},
		{T: "status"},
		{T: "delta", Text: "Install it "},
		{T: "delta", Text: "with one line."},
	}}
	h := askHandler(site, map[string]Provider{"fake": fake}, "sys", &Ledger{path: filepath.Join(t.TempDir(), "l.jsonl")})
	body := `{"provider":"fake","history":[{"role":"user","text":"how do I install it?"}]}`
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodPost, "/api/ask", strings.NewReader(body)))

	if ct := rec.Header().Get("Content-Type"); ct != "application/x-ndjson" {
		t.Fatalf("content type %q", ct)
	}
	var got []Event
	for _, line := range strings.Split(strings.TrimSpace(rec.Body.String()), "\n") {
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("not one JSON event per line: %q", line)
		}
		got = append(got, e)
	}
	kinds := []string{}
	for _, e := range got {
		kinds = append(kinds, e.T)
	}
	if strings.Join(kinds, ",") != "delta,reset,status,delta,delta,done" {
		t.Fatalf("events %v", kinds)
	}
	if got[2].Page != "https://it-rat.com/install.html" {
		t.Fatalf("status names %q", got[2].Page)
	}
	d := got[len(got)-1].Done
	if d == nil || d.Answer != "Install it with one line." || d.Provider != "fake" || len(d.PagesRead) != 1 {
		t.Fatalf("done %+v", d)
	}
	if d.FirstMs < 0 || d.FirstMs > d.Ms {
		t.Fatalf("first_ms %d outside 0..ms %d", d.FirstMs, d.Ms)
	}
}

func TestAskRefusesBadRequestsBeforeStreaming(t *testing.T) {
	h := askHandler(fixtureSite(t), map[string]Provider{"fake": fakeProvider{}}, "sys", &Ledger{path: filepath.Join(t.TempDir(), "l.jsonl")})
	for name, body := range map[string]string{
		"unknown provider": `{"provider":"free","history":[{"role":"user","text":"hi"}]}`,
		"ends on model":    `{"provider":"fake","history":[{"role":"user","text":"hi"},{"role":"assistant","text":"yo"}]}`,
		"not json":         `{`,
	} {
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodPost, "/api/ask", strings.NewReader(body)))
		if rec.Code != http.StatusBadRequest || rec.Header().Get("Content-Type") != "application/json" {
			t.Errorf("%s: code %d type %q", name, rec.Code, rec.Header().Get("Content-Type"))
		}
	}
}

// A probe asking for a comparison with another product lost its whole answer
// on 2026-11-01: the model wrote it,
// then called contact_team, and the text was dropped as narration.
func TestKeepTurnText(t *testing.T) {
	cases := []struct {
		called []string
		want   bool
	}{
		{nil, true},
		{[]string{"contact_team"}, true},
		{[]string{"read_page"}, false},
		{[]string{"read_page", "contact_team"}, false},
		{[]string{"contact_team", "read_page"}, false},
	}
	for _, c := range cases {
		if got := keepTurnText(c.called); got != c.want {
			t.Errorf("keepTurnText(%v) = %v, want %v", c.called, got, c.want)
		}
	}
}

func TestHaikuCost(t *testing.T) {
	// 20k fresh input + 6k cache write + 6k cache read + 1k out
	got := haikuCost(Usage{In: 20000, CacheWrite: 6000, CacheRead: 6000, Out: 1000})
	want := (20000*0.10 + 6000*0.125 + 6000*0.01 + 1000*0.50) / 1e6
	if d := got - want; d > 1e-12 || d < -1e-12 {
		t.Fatalf("cost %v want %v", got, want)
	}
}
