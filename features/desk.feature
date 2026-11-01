# The desk assistant: what a visitor gets, and what it will not do.
#
# Provenance. No sentence below is a quote.
#
#   @decided 2026-11-01   A visitor's question about the site is answered by a model reading
#                         the site's own pages. The chat carries no buttons; writing to the team
#                         lives on the team's address beside the chat. The answer is shown from
#                         its first line, and it streams. Haiku 5.5 is the model.
#   @decided 2026-11-01   The prototype is kept, working, and not wired into the live site: no
#                         relay on the internet now. The reason was that a public assistant can
#                         be used for anything, and the site has no traffic yet to justify it.
#   @claude 2026-11-01    How each rule holds is mine: the tests named below.
#
# Each scenario names the test that holds it with "# -> TestName"; TestScenariosAreBound
# checks that link both ways, and scripts/gates-have-teeth.sh proves that check can fail.

Feature: A desk assistant that answers from the site and nothing else

  Scenario: a question is answered from a page the site lists, as it is written
    Given the site's index lists install.md
    When a visitor asks how to install the stack
    Then the page being read is shown first, the answer arrives word by word,
    And a final line carries the whole answer and the page it came from
    # -> TestAskStreamsInOrderAndEndsWithDone

  Scenario: words the model writes before reading are taken back
    Given the model says "let me check" and then reads a page
    When the answer is assembled
    Then the narration is not part of it, while an answer written before offering the team is
    # -> TestKeepTurnText

  Scenario: only pages the site lists can be read
    Given an index that lists two pages and a third that is gone
    When the server loads the site
    Then exactly the two pages present on disk are readable
    # -> TestAllowlistIsExactlyTheListedPagesOnDisk

  Scenario: a listed page is found however the model spells it
    Given install.md is listed
    When the model asks for it with or without the origin, a leading slash or .html
    Then the same page is read and cited under its human URL
    # -> TestReadPageAcceptsListedForms

  Scenario: internal files and escapes are refused
    Given CLAUDE.md, SEO-STATE.md and llms-full.txt sit beside the pages
    When the model asks for them, or for a path with .. or a NUL in it
    Then nothing is read
    # -> TestReadPageRefusesEverythingElse

  Scenario: a site that lists no page does not start an empty assistant
    Given an index that lists nothing, or no index at all
    When the server loads the site
    Then it refuses to start
    # -> TestLoadSiteRefusesAnIndexWithNoPages

  Scenario: the conversation a browser may send has one shape
    Given turns that alternate from the visitor, at most twenty, each at most 2000 characters
    When anything else arrives: a system turn, two visitor turns in a row, a blank or an oversized turn
    Then it is refused
    # -> TestValidateHistory

  Scenario: a malformed request is refused before the model is asked
    Given a request naming no known model, ending on the assistant, or not JSON at all
    When it reaches the server
    Then it gets a plain 400 and nothing streams
    # -> TestAskRefusesBadRequestsBeforeStreaming

  Scenario: asking for a person prepares the note to the team
    Given a visitor who wants to talk about a pilot
    When the model hands the request to the team
    Then the note form behind the team's address is pre-filled with a one-line summary
    And the model cannot use that moment to read an internal file
    # -> TestContactToolSetsTheFlag

  Scenario: the key stays on this machine
    Given the server holds the API key
    When it is told to listen anywhere but loopback
    Then it refuses to start
    # -> TestLoopbackOnly

  Scenario: every question is priced in the ledger
    Given Haiku 5.5's list prices, cache writes at 1.25 times and cache reads at 0.1 times input
    When a question used fresh, cached and written tokens
    Then the ledger records what they cost
    # -> TestHaikuCost
