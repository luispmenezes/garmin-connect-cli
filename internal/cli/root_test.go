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
	} {
		if _, _, err := root.Find(args); err != nil {
			t.Fatalf("%v missing: %v", args, err)
		}
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
