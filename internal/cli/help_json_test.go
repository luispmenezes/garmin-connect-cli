package cli

import (
	"encoding/json"
	"testing"
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
