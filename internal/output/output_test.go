package output

import (
	"bytes"
	"testing"
)

func TestWriteJSONCompactAndPretty(t *testing.T) {
	var compact bytes.Buffer
	if err := (Options{Stdout: &compact}).WriteJSON([]byte(`{"b":2,"a":1}`)); err != nil {
		t.Fatal(err)
	}
	if compact.String() != "{\"a\":1,\"b\":2}\n" {
		t.Fatalf("compact = %q", compact.String())
	}
	var pretty bytes.Buffer
	if err := (Options{Stdout: &pretty, Pretty: true}).WriteJSON([]byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	if pretty.String() != "{\n  \"a\": 1\n}\n" {
		t.Fatalf("pretty = %q", pretty.String())
	}
}
