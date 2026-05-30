package cli

import (
	"encoding/json"
	"strings"

	"github.com/luispmenezes/garmin-connect-cli/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type commandDoc struct {
	Command     string    `json:"command"`
	Use         string    `json:"use"`
	Short       string    `json:"short,omitempty"`
	Long        string    `json:"long,omitempty"`
	Aliases     []string  `json:"aliases,omitempty"`
	Args        string    `json:"args,omitempty"`
	Flags       []flagDoc `json:"flags,omitempty"`
	GlobalFlags []flagDoc `json:"global_flags,omitempty"`
	Examples    []string  `json:"examples,omitempty"`
	Children    []string  `json:"children,omitempty"`
}

type flagDoc struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Usage     string `json:"usage"`
	Default   string `json:"default,omitempty"`
	Type      string `json:"type"`
	Required  bool   `json:"required,omitempty"`
}

func (a *app) helpJSONCommand(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "help-json [COMMAND...]",
		Short: "Print machine-readable command help",
		Long:  "Print command metadata as JSON for scripts and AI agents.",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := root
			if len(args) > 0 {
				found, _, err := root.Find(args)
				if err != nil {
					return err
				}
				target = found
			}
			docs := collectDocs(target)
			return a.out().WriteValue(map[string]any{
				"binary":      "garmin",
				"version":     version.Version,
				"output":      "JSON is the stable scripting contract. Errors, prompts, and progress go to stderr.",
				"commands":    docs,
				"unavailable": []string{"sync", "devices history"},
			})
		},
	}
}

func collectDocs(root *cobra.Command) []commandDoc {
	var docs []commandDoc
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		if cmd.Hidden {
			return
		}
		docs = append(docs, docForCommand(cmd))
		for _, child := range cmd.Commands() {
			if child.IsAvailableCommand() || child.Name() == "help" {
				walk(child)
			}
		}
	}
	walk(root)
	return docs
}

func docForCommand(cmd *cobra.Command) commandDoc {
	children := make([]string, 0)
	for _, child := range cmd.Commands() {
		if child.IsAvailableCommand() || child.Name() == "help" {
			children = append(children, child.Name())
		}
	}
	examples := splitExamples(cmd.Example)
	return commandDoc{
		Command:     cmd.CommandPath(),
		Use:         cmd.UseLine(),
		Short:       cmd.Short,
		Long:        strings.TrimSpace(cmd.Long),
		Aliases:     cmd.Aliases,
		Args:        argsDescription(cmd),
		Flags:       flags(cmd.NonInheritedFlags()),
		GlobalFlags: flags(cmd.InheritedFlags()),
		Examples:    examples,
		Children:    children,
	}
}

func flags(set *pflag.FlagSet) []flagDoc {
	out := make([]flagDoc, 0)
	if set == nil {
		return out
	}
	set.VisitAll(func(f *pflag.Flag) {
		required := false
		if v := f.Annotations[cobra.BashCompOneRequiredFlag]; len(v) > 0 {
			required = true
		}
		out = append(out, flagDoc{
			Name:      f.Name,
			Shorthand: f.Shorthand,
			Usage:     f.Usage,
			Default:   f.DefValue,
			Type:      f.Value.Type(),
			Required:  required,
		})
	})
	return out
}

func splitExamples(s string) []string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func argsDescription(cmd *cobra.Command) string {
	if strings.Contains(cmd.Use, " ") {
		return strings.TrimSpace(strings.TrimPrefix(cmd.Use, cmd.Name()))
	}
	return ""
}

func marshalCommandDocForTest(cmd *cobra.Command) ([]byte, error) {
	return json.Marshal(docForCommand(cmd))
}
