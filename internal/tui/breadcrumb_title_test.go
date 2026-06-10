package tui

import (
	"strings"
	"testing"
)

// TestRightPaneTitleSessions verifies that opening a session preview places a
// title for the right pane in the title bar, positioned above the right pane.
func TestRightPaneTitleSessions(t *testing.T) {
	app := newTestApp(fakeSessions())
	app.sessSplit.Show = true
	app.sessPreviewMode = sessPreviewStats

	if got := app.rightPaneTitle(); got != "Stats" {
		t.Fatalf("rightPaneTitle = %q, want %q", got, "Stats")
	}

	bar := stripANSI(app.renderBreadcrumb())
	dividerCol := app.activeDividerCol()
	if dividerCol <= 0 {
		t.Fatalf("expected a visible split, dividerCol=%d", dividerCol)
	}
	if !strings.Contains(bar, "Stats") {
		t.Fatalf("title bar missing right-pane title: %q", bar)
	}
	// The title should sit above the right pane (at or after the divider column).
	idx := strings.Index(bar, "Stats")
	if idx < dividerCol {
		t.Fatalf("right-pane title at col %d, want >= dividerCol %d: %q", idx, dividerCol, bar)
	}
}

// TestRightPaneTitleHiddenWithoutSplit verifies the title is absent when no
// preview pane is open.
func TestRightPaneTitleHiddenWithoutSplit(t *testing.T) {
	app := newTestApp(fakeSessions())
	app.sessSplit.Show = false

	if got := app.rightPaneTitle(); got != "" {
		t.Fatalf("rightPaneTitle = %q, want empty when no split", got)
	}
}

func TestSessPreviewTitleLabels(t *testing.T) {
	cases := map[sessPreview]string{
		sessPreviewConversation: "Conversation",
		sessPreviewStats:        "Stats",
		sessPreviewMemory:       "Memory",
		sessPreviewTasksPlan:    "Tasks / Plan",
		sessPreviewAgents:       "Agents",
		sessPreviewContexts:     "Contexts",
		sessPreviewLive:         "Live",
	}
	for mode, want := range cases {
		if got := sessPreviewTitle(mode); got != want {
			t.Errorf("sessPreviewTitle(%d) = %q, want %q", mode, got, want)
		}
	}
}
