package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRemovedCommandsUnavailable(t *testing.T) {
	root := NewRootCommand()
	if findSubcommand(root, "sync") != nil {
		t.Fatal("sync command should not exist")
	}
	devices := findSubcommand(root, "devices")
	if devices == nil {
		t.Fatal("devices command missing")
	}
	if findSubcommand(devices, "history") != nil {
		t.Fatal("devices history command should not exist")
	}
}

func TestExpectedCommandsAvailable(t *testing.T) {
	root := NewRootCommand()
	for _, args := range [][]string{
		{"auth", "login"},
		{"profile", "show"},
		{"activities", "download"},
		{"devices", "list"},
		{"health", "training-readiness"},
		{"health", "weight-add"},
		{"version"},
	} {
		if _, _, err := root.Find(args); err != nil {
			t.Fatalf("%v missing: %v", args, err)
		}
	}
}

func TestWeightAddValidatesArgsBeforeAuth(t *testing.T) {
	root := NewRootCommand()
	root.SetArgs([]string{"health", "weight-add"})
	err := root.Execute()
	if err == nil || err.Error() != `accepts 1 arg(s), received 0` {
		t.Fatalf("expected missing argument error, got %v", err)
	}
}

func TestApplyDisplayNameEscapesPathSegment(t *testing.T) {
	got := applyDisplayName("/wellness/{displayName}/daily", "Luis Menezes/QA")
	want := "/wellness/Luis%20Menezes%2FQA/daily"
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func findSubcommand(cmd commandLister, name string) commandLister {
	for _, child := range cmd.Commands() {
		if child.Name() == name {
			return child
		}
	}
	return nil
}

type commandLister interface {
	Name() string
	Commands() []*cobra.Command
}
