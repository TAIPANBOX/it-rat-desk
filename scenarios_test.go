package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// unbound lists the tests no scenario has to claim, each with its reason.
var unbound = map[string]string{
	"TestScenariosAreBound": "this check itself",
}

// TestScenariosAreBound holds the link between features/*.feature and the
// tests, both ways. Forward: every scenario names exactly one test with a
// "# -> TestName" line, and that test exists. Backward: every test is named by
// some scenario, unless unbound says why not. The backward half is the one that
// catches drift: without it tests accumulate that no sentence describes, and the
// feature file still reads as the whole behaviour.
func TestScenariosAreBound(t *testing.T) {
	feats, _ := filepath.Glob("features/*.feature")
	if len(feats) == 0 {
		t.Fatal("measured nothing: no feature file under features/")
	}
	tests := map[string]bool{}
	srcs, _ := filepath.Glob("*_test.go")
	decl := regexp.MustCompile(`(?m)^func (Test\w+)\(t \*testing\.T\)`)
	for _, src := range srcs {
		b, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range decl.FindAllStringSubmatch(string(b), -1) {
			tests[m[1]] = true
		}
	}
	claimed := map[string]bool{}
	scenarios := 0
	for _, f := range feats {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		scenario, named := "", 0
		flush := func() {
			if scenario != "" && named != 1 {
				t.Errorf("%s: scenario %q names %d tests, want exactly one", f, scenario, named)
			}
		}
		for _, line := range strings.Split(string(b), "\n") {
			s := strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(s, "Scenario:"):
				flush()
				scenario, named = strings.TrimSpace(strings.TrimPrefix(s, "Scenario:")), 0
				scenarios++
			case strings.HasPrefix(s, "# -> ") && scenario != "":
				name := strings.TrimSpace(strings.TrimPrefix(s, "# -> "))
				named++
				if !tests[name] {
					t.Errorf("%s: scenario %q names %s, which is no test", f, scenario, name)
				}
				claimed[name] = true
			}
		}
		flush()
	}
	if scenarios == 0 {
		t.Fatal("measured nothing: the feature files hold no scenario")
	}
	for name := range tests {
		if !claimed[name] && unbound[name] == "" {
			t.Errorf("%s is claimed by no scenario", name)
		}
	}
}
