package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type Options struct {
	Format string
	Pretty bool
	Stdout io.Writer
}

func (o Options) WriteJSON(data []byte) error {
	out := o.Stdout
	if out == nil {
		out = io.Discard
	}
	if len(data) == 0 {
		_, err := fmt.Fprintln(out, "null")
		return err
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var encoded []byte
	var err error
	if o.Pretty {
		encoded, err = json.MarshalIndent(raw, "", "  ")
	} else {
		encoded, err = json.Marshal(raw)
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(encoded))
	return err
}

func (o Options) WriteValue(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return o.WriteJSON(data)
}

func (o Options) WriteTable(headers []string, rows [][]string) error {
	out := o.Stdout
	if out == nil {
		out = io.Discard
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	return w.Flush()
}

func Truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
