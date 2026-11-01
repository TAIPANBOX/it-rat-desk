#!/bin/sh
# gates-have-teeth.sh: plants each fault TestScenariosAreBound exists to catch,
# in a throwaway copy of the repository, and requires the check to fail on it
# for the right reason. It also requires the check to pass on a harmless edit,
# and to fail saying it measured nothing when its subject is gone.
#
# A case expecting failure proves nothing if the check was already failing, so
# the unmutated copy must pass first; otherwise the run stops as UNJUDGEABLE.
set -eu

ROOT=$(cd "$(dirname "$0")/.." && pwd)
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT INT TERM

fresh() {
	rm -rf "$WORK/t"
	mkdir -p "$WORK/t"
	(cd "$ROOT" && tar --exclude .git --exclude testdata -cf - .) | (cd "$WORK/t" && tar -xf -)
}
check() {
	(cd "$WORK/t" && go test -count=1 -run '^TestScenariosAreBound$' . 2>&1)
}

fresh
if ! check >/dev/null; then
	echo "UNJUDGEABLE: TestScenariosAreBound fails on the unmutated tree"
	exit 1
fi

cases=0
failures=0
# expect <fail|pass> <name> <needle> <mutation, run inside the copy>
expect() {
	want=$1 name=$2 needle=$3 mutation=$4
	cases=$((cases + 1))
	fresh
	(cd "$WORK/t" && sh -c "$mutation") || { echo "BROKEN      $name: its mutation did not apply"; failures=$((failures + 1)); return; }
	if out=$(check); then rc=0; else rc=1; fi
	if [ "$want" = fail ] && [ $rc -eq 0 ]; then
		echo "TOOTHLESS   $name"
		failures=$((failures + 1))
	elif [ "$want" = fail ] && ! printf '%s' "$out" | grep -qF -- "$needle"; then
		echo "WRONG REASON $name: it failed, but not saying: $needle"
		failures=$((failures + 1))
	elif [ "$want" = pass ] && [ $rc -ne 0 ]; then
		echo "OVEREAGER   $name"
		failures=$((failures + 1))
	else
		echo "ok  $name ($want)"
	fi
}

expect fail "a scenario names a test that does not exist" "which is no test" \
	'grep -q "# -> TestValidateHistory" features/desk.feature && perl -pi -e "s/# -> TestValidateHistory/# -> TestNoSuchTest/" features/desk.feature'

expect fail "a test no scenario describes" "claimed by no scenario" \
	'printf "\nfunc TestOrphan(t *testing.T) {}\n" >> main_test.go'

expect fail "a scenario that names no test" "names 0 tests" \
	'grep -q "# -> TestHaikuCost" features/desk.feature && perl -ni -e "print unless /# -> TestHaikuCost/" features/desk.feature'

expect fail "a scenario that names two tests" "names 2 tests" \
	'perl -pi -e "s/(# -> TestLoopbackOnly)/\$1\n    # -> TestHaikuCost/" features/desk.feature && grep -c "# -> TestHaikuCost" features/desk.feature | grep -q 2'

expect fail "the feature files are gone" "measured nothing" \
	'rm -rf features'

expect fail "a feature file with no scenario in it" "measured nothing" \
	'printf "Feature: empty\n" > features/desk.feature'

expect pass "a comment added to a feature file (non-fault)" "" \
	'printf "\n# a note for the reader\n" >> features/desk.feature'

if [ $failures -ne 0 ]; then
	echo "FAIL: $failures of $cases cases"
	exit 1
fi
echo "OK: $cases cases. The scenario binding fails on each fault, passes on a non-fault,"
echo "    and refuses to report success when it measured nothing."
