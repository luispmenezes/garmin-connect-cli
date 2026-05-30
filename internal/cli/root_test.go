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
		{"workouts", "list"},
		{"workouts", "get"},
		{"workouts", "create"},
		{"workouts", "update"},
		{"workouts", "delete"},
		{"workouts", "schedule"},
		{"workouts", "schedule-get"},
		{"workouts", "unschedule"},
		{"calendar", "workouts"},
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

func TestWorkoutCommandsValidateArgsBeforeAuth(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "get missing id",
			args:    []string{"workouts", "get"},
			wantErr: `accepts 1 arg(s), received 0`,
		},
		{
			name:    "schedule invalid date",
			args:    []string{"workouts", "schedule", "123", "--date", "2026-02-31"},
			wantErr: `invalid date "2026-02-31"; use YYYY-MM-DD`,
		},
		{
			name:    "calendar invalid start date",
			args:    []string{"calendar", "workouts", "--start", "2026/06/01"},
			wantErr: `invalid date "2026/06/01"; use YYYY-MM-DD`,
		},
		{
			name:    "create missing file",
			args:    []string{"workouts", "create", "testdata/does-not-exist.json"},
			wantErr: `open testdata/does-not-exist.json: no such file or directory`,
		},
		{
			name:    "update missing file",
			args:    []string{"workouts", "update", "123", "testdata/does-not-exist.json"},
			wantErr: `open testdata/does-not-exist.json: no such file or directory`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := NewRootCommand()
			root.SetArgs(tt.args)
			err := root.Execute()
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("expected %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateDate(t *testing.T) {
	if err := validateDate("2026-06-01"); err != nil {
		t.Fatalf("valid date rejected: %v", err)
	}
	if err := validateDate(""); err == nil || err.Error() != "date is required in YYYY-MM-DD format" {
		t.Fatalf("expected required date error, got %v", err)
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
