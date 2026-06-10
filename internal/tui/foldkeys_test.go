package tui

import (
	"testing"

	"github.com/sendbird/ccx/internal/session"
)

// foldTestEntry returns an entry with a collapsible (tool_use) block plus a
// text block, so fold-all has a visible effect.
func foldTestEntry() session.Entry {
	return session.Entry{
		Role: "assistant",
		Content: []session.ContentBlock{
			{Type: "text", Text: "hello"},
			{Type: "tool_use", ToolName: "Bash", ToolInput: "ls"},
		},
	}
}

func TestFoldStateHonorsConfiguredFoldAllKey(t *testing.T) {
	fs := &FoldState{Collapsed: make(foldSet), Entry: foldTestEntry()}
	fs.applyPreviewKeys(Keymap{Preview: PreviewKeymap{FoldAll: "z", ExpandAll: "Z"}})

	// Default "f" must no longer fold once the binding is customized.
	if got := fs.HandleKey("f"); got != foldUnhandled {
		t.Fatalf(`"f" should be unhandled when FoldAll is rebound, got %v`, got)
	}

	// The configured key folds (collapses the tool_use block).
	if got := fs.HandleKey("z"); got != foldHandled {
		t.Fatalf(`"z" should fold-all, got %v`, got)
	}
	if len(fs.Collapsed) == 0 {
		t.Fatal("fold-all should collapse the tool_use block")
	}

	// The configured expand key unfolds everything.
	if got := fs.HandleKey("Z"); got != foldHandled {
		t.Fatalf(`"Z" should expand-all, got %v`, got)
	}
	if len(fs.Collapsed) != 0 {
		t.Fatalf("expand-all should clear collapsed set, got %d", len(fs.Collapsed))
	}
}

func TestFoldStateDefaultsToFAndShiftF(t *testing.T) {
	fs := &FoldState{Collapsed: make(foldSet), Entry: foldTestEntry()}
	// No applyPreviewKeys: empty keys fall back to "f"/"F".
	if got := fs.HandleKey("f"); got != foldHandled {
		t.Fatalf(`default "f" should fold-all, got %v`, got)
	}
	if len(fs.Collapsed) == 0 {
		t.Fatal("default fold-all should collapse blocks")
	}
	if got := fs.HandleKey("F"); got != foldHandled {
		t.Fatalf(`default "F" should expand-all, got %v`, got)
	}
	if len(fs.Collapsed) != 0 {
		t.Fatal("default expand-all should clear collapsed set")
	}
}
