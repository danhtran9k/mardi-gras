package app

import (
	"sort"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/matt-wright86/mardi-gras/internal/data"
	"github.com/matt-wright86/mardi-gras/internal/gastown"
)

// closedBlockerIssues: "done" is closed; "wip" (in progress) and "next"
// (open) are blocked only by it, so both are free to move. "waiting" is
// blocked by the still-open "next".
func closedBlockerIssues() []data.Issue {
	blocks := func(id, on string) []data.Dependency {
		return []data.Dependency{{IssueID: id, DependsOnID: on, Type: "blocks"}}
	}
	done := testIssue("done", data.StatusClosed)
	wip := testIssue("wip", data.StatusInProgress)
	wip.Assignee = "alice"
	wip.Dependencies = blocks("wip", "done")
	next := testIssue("next", data.StatusOpen)
	next.Dependencies = blocks("next", "done")
	waiting := testIssue("waiting", data.StatusOpen)
	waiting.Dependencies = blocks("waiting", "next")
	return []data.Issue{done, wip, next, waiting}
}

func closedBlockerModel(t *testing.T, filters ...Filters) Model {
	t.Helper()
	t.Setenv("BEADS_ACTOR", "alice")
	m := New(closedBlockerIssues(), data.Source{}, data.DefaultBlockingTypes, filters...)
	m.startedAt = time.Now().Add(-time.Second)
	m.driver = gastown.NewGTDriver()
	model, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	return model.(Model)
}

func groupIDs(groups map[data.ParadeStatus][]data.Issue, s data.ParadeStatus) []string {
	var ids []string
	for _, iss := range groups[s] {
		ids = append(ids, iss.ID)
	}
	sort.Strings(ids)
	return ids
}

func assertGroup(t *testing.T, groups map[data.ParadeStatus][]data.Issue, s data.ParadeStatus, want ...string) {
	t.Helper()
	got := groupIDs(groups, s)
	if len(got) != len(want) {
		t.Fatalf("group %v = %v, want %v", s, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("group %v = %v, want %v", s, got, want)
		}
	}
}

// A closed blocker is resolved in every view, even when the view hides it.
func TestFocusModeClosedBlockerIsResolved(t *testing.T) {
	m := closedBlockerModel(t)
	assertGroup(t, m.header.Groups, data.ParadeRolling, "wip")
	assertGroup(t, m.header.Groups, data.ParadeStalled, "waiting")

	m.focusMode = true
	m.rebuildParade()
	g := m.header.Groups
	assertGroup(t, g, data.ParadeRolling, "wip")
	assertGroup(t, g, data.ParadeLinedUp, "next")
	assertGroup(t, g, data.ParadeStalled, "waiting")
}

func TestFilterClosedBlockerIsResolved(t *testing.T) {
	m := closedBlockerModel(t)
	m.filterInput.SetValue("wip")
	m.rebuildParade()
	assertGroup(t, m.header.Groups, data.ParadeRolling, "wip")
	assertGroup(t, m.header.Groups, data.ParadeStalled)
}

// Startup --exclude-type/--exclude-label and --status group a subset too.
func TestGroupSubsetClosedBlockerIsResolved(t *testing.T) {
	issues := closedBlockerIssues()
	visible := []data.Issue{issues[1], issues[2]} // "done" and "waiting" hidden
	g := data.GroupByParadeWith(visible, data.BuildIssueMap(issues), data.DefaultBlockingTypes)
	assertGroup(t, g, data.ParadeRolling, "wip")
	assertGroup(t, g, data.ParadeLinedUp, "next")
	assertGroup(t, g, data.ParadeStalled)
}
