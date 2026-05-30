package cli

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/luispmenezes/garmin-connect-cli/internal/version"
	"github.com/spf13/cobra"
)

func TestHelpJSONIncludesAgentMetadata(t *testing.T) {
	root := NewRootCommand()
	cmd, _, err := root.Find([]string{"activities", "download"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := marshalCommandDocForTest(cmd)
	if err != nil {
		t.Fatal(err)
	}
	var doc commandDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Command != "garmin activities download" {
		t.Fatalf("command = %q", doc.Command)
	}
	var foundOutput bool
	for _, flag := range doc.Flags {
		if flag.Name == "output" {
			foundOutput = true
			if !flag.Required {
				t.Fatal("output flag should be marked required")
			}
		}
	}
	if !foundOutput {
		t.Fatal("output flag missing")
	}
}

func TestHelpJSONUsesInjectedVersion(t *testing.T) {
	previous := version.Version
	version.Version = "9.8.7-test"
	t.Cleanup(func() { version.Version = previous })

	var out bytes.Buffer
	a := &app{stdout: &out}
	root := &cobra.Command{Use: "garmin"}
	root.AddCommand(a.helpJSONCommand(root))
	cmd, _, err := root.Find([]string{"help-json"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	var doc struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Version != "9.8.7-test" {
		t.Fatalf("version = %q", doc.Version)
	}
}
