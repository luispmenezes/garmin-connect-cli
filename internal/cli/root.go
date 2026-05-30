package cli

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/luispmenezes/garmin-connect-cli/internal/auth"
	"github.com/luispmenezes/garmin-connect-cli/internal/garmin"
	"github.com/luispmenezes/garmin-connect-cli/internal/output"
	"github.com/luispmenezes/garmin-connect-cli/internal/version"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type app struct {
	profile string
	format  string
	pretty  bool
	stdout  io.Writer
	stderr  io.Writer
}

func NewRootCommand() *cobra.Command {
	a := &app{stdout: os.Stdout, stderr: os.Stderr}
	root := &cobra.Command{
		Use:           "garmin",
		Short:         "Garmin Connect command-line client",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&a.profile, "profile", "", "credential profile (defaults to GARMIN_PROFILE or default)")
	root.PersistentFlags().StringVar(&a.format, "format", "json", "output format: json or table")
	root.PersistentFlags().BoolVar(&a.pretty, "pretty", false, "pretty-print JSON output")
	root.AddCommand(a.authCommand(), a.profileCommand(), a.activitiesCommand(), a.devicesCommand(), a.healthCommand(), a.versionCommand())
	root.AddCommand(a.helpJSONCommand(root))
	return root
}

func (a *app) out() output.Options {
	return output.Options{Format: a.format, Pretty: a.pretty, Stdout: a.stdout}
}

func (a *app) store() (*auth.Store, error) {
	return auth.NewStore(a.profile)
}

func (a *app) api() (*garmin.Client, auth.OAuth2Token, error) {
	store, err := a.store()
	if err != nil {
		return nil, auth.OAuth2Token{}, err
	}
	_, token, err := auth.EnsureFresh(store)
	if err != nil {
		return nil, auth.OAuth2Token{}, err
	}
	return garmin.NewClient(), token, nil
}

func (a *app) versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.out().WriteValue(version.Get())
		},
	}
}

func (a *app) writeRawOrTable(data []byte, table func([]byte) ([][]string, []string, error)) error {
	if a.format == "json" {
		return a.out().WriteJSON(data)
	}
	if a.format != "table" {
		return fmt.Errorf("unsupported output format %q", a.format)
	}
	rows, headers, err := table(data)
	if err != nil {
		return err
	}
	return a.out().WriteTable(headers, rows)
}

func (a *app) authCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Manage Garmin authentication"}
	var email string
	login := &cobra.Command{
		Use:     "login",
		Short:   "Log in to Garmin Connect",
		Example: "garmin auth login --email user@example.com",
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				fmt.Fprint(a.stderr, "Email: ")
				if _, err := fmt.Fscanln(os.Stdin, &email); err != nil {
					return err
				}
			}
			fmt.Fprint(a.stderr, "Password: ")
			passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(a.stderr)
			if err != nil {
				return err
			}
			client, err := auth.NewSSOClient()
			if err != nil {
				return err
			}
			oauth1, oauth2, err := client.Login(email, string(passwordBytes), func(method string) (string, error) {
				fmt.Fprintf(a.stderr, "MFA code (%s): ", method)
				var code string
				_, err := fmt.Fscanln(os.Stdin, &code)
				return code, err
			})
			if err != nil {
				return err
			}
			store, err := a.store()
			if err != nil {
				return err
			}
			if err := store.SaveTokens(oauth1, oauth2); err != nil {
				return err
			}
			return a.out().WriteValue(map[string]any{"authenticated": true, "profile": store.Profile(), "expires_at": oauth2.ExpiresAt})
		},
	}
	login.Flags().StringVar(&email, "email", "", "Garmin account email")
	status := &cobra.Command{
		Use:     "status",
		Short:   "Show authentication status",
		Example: "garmin auth status --pretty",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.store()
			if err != nil {
				return err
			}
			_, oauth2, ok, err := store.LoadTokens()
			if err != nil {
				return err
			}
			status := map[string]any{"authenticated": ok, "profile": store.Profile()}
			if ok {
				status["expired"] = oauth2.Expired(time.Now())
				status["expires_at"] = oauth2.ExpiresAt
			}
			return a.out().WriteValue(status)
		},
	}
	logout := &cobra.Command{
		Use:     "logout",
		Short:   "Remove stored tokens",
		Example: "garmin auth logout",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := a.store()
			if err != nil {
				return err
			}
			if err := store.Clear(); err != nil {
				return err
			}
			return a.out().WriteValue(map[string]any{"authenticated": false, "profile": store.Profile()})
		},
	}
	cmd.AddCommand(login, status, logout)
	return cmd
}

func (a *app) profileCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "profile", Short: "Show Garmin profile data"}
	cmd.AddCommand(a.getCommand("show", "Show social profile", garmin.ProfilePath(), profileTable))
	cmd.AddCommand(a.getCommand("settings", "Show user settings", garmin.SettingsPath(), genericKVTable))
	return cmd
}

func (a *app) devicesCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "devices", Short: "Show Garmin devices"}
	cmd.AddCommand(a.getCommand("list", "List devices", garmin.DeviceListPath(), devicesTable))
	get := &cobra.Command{
		Use:   "get DEVICE_ID",
		Short: "Show device details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, token, err := a.api()
			if err != nil {
				return err
			}
			data, err := client.GetJSON(token, garmin.DeviceGetPath(args[0]))
			if err != nil {
				return err
			}
			return a.writeRawOrTable(data, genericKVTable)
		},
	}
	cmd.AddCommand(get)
	return cmd
}

func (a *app) activitiesCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "activities", Short: "Manage Garmin activities"}
	var limit, start int
	list := &cobra.Command{
		Use:   "list",
		Short: "List activities",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, token, err := a.api()
			if err != nil {
				return err
			}
			data, err := client.GetJSON(token, garmin.ActivityListPath(start, limit))
			if err != nil {
				return err
			}
			return a.writeRawOrTable(data, activitiesTable)
		},
	}
	list.Flags().IntVar(&limit, "limit", 20, "number of activities to return")
	list.Flags().IntVar(&start, "start", 0, "activity offset")
	get := &cobra.Command{
		Use:   "get ACTIVITY_ID",
		Short: "Show activity details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, token, err := a.api()
			if err != nil {
				return err
			}
			data, err := client.GetJSON(token, garmin.ActivityGetPath(args[0]))
			if err != nil {
				return err
			}
			return a.writeRawOrTable(data, genericKVTable)
		},
	}
	var downloadType, downloadOutput string
	download := &cobra.Command{
		Use:   "download ACTIVITY_ID",
		Short: "Download activity file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if downloadOutput == "" {
				return errors.New("--output is required; use --output - to stream to stdout")
			}
			endpoint, err := garmin.ActivityDownloadEndpoint(args[0], strings.ToLower(downloadType))
			if err != nil {
				return err
			}
			client, token, err := a.api()
			if err != nil {
				return err
			}
			data, err := client.Download(token, endpoint.Path)
			if err != nil {
				return err
			}
			if downloadOutput == "-" {
				_, err = a.stdout.Write(data)
				return err
			}
			if err := os.WriteFile(downloadOutput, data, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(a.stderr, "wrote %s (%d bytes)\n", downloadOutput, len(data))
			_ = endpoint.Extension
			return nil
		},
	}
	download.Flags().StringVar(&downloadType, "type", "fit", "download type: fit, gpx, tcx, or kml")
	download.Flags().StringVarP(&downloadOutput, "output", "o", "", "output path, or - for stdout")
	_ = download.MarkFlagRequired("output")
	upload := &cobra.Command{
		Use:   "upload FILE",
		Short: "Upload an activity file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, token, err := a.api()
			if err != nil {
				return err
			}
			data, err := client.Upload(token, "/upload-service/upload/.fit", args[0])
			if err != nil {
				return err
			}
			return a.out().WriteJSON(data)
		},
	}
	cmd.AddCommand(list, get, download, upload)
	return cmd
}

func (a *app) healthCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "health", Short: "Show Garmin health metrics"}
	for _, spec := range healthSpecs() {
		spec := spec
		cmd.AddCommand(a.healthMetricCommand(spec))
	}
	return cmd
}

func (a *app) healthMetricCommand(spec healthSpec) *cobra.Command {
	var date, from, to string
	var days int
	var unit string
	cmd := &cobra.Command{
		Use:   spec.use,
		Short: spec.short,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, token, err := a.api()
			if err != nil {
				return err
			}
			path, err := spec.path(date, from, to, days, args)
			if err != nil {
				return err
			}
			if strings.Contains(path, "{displayName}") {
				displayName, err := getDisplayName(client, token)
				if err != nil {
					return err
				}
				path = applyDisplayName(path, displayName)
			}
			if spec.post {
				if len(args) != 1 {
					return errors.New("weight-add requires WEIGHT")
				}
				weight, err := strconv.ParseFloat(args[0], 64)
				if err != nil {
					return fmt.Errorf("invalid weight %q", args[0])
				}
				grams, err := weightGrams(weight, unit)
				if err != nil {
					return err
				}
				now := time.Now()
				calendarDate := dateOrToday(date)
				timestamp := now.UnixMilli()
				body := map[string]any{
					"dateTimestamp": timestamp,
					"gmtTimestamp":  timestamp,
					"weight":        grams,
					"unitKey":       "kg",
					"calendarDate":  calendarDate,
				}
				data, err := client.PostJSON(token, path, body)
				if err != nil {
					return err
				}
				return a.out().WriteJSON(data)
			}
			data, err := client.GetJSON(token, path)
			if err != nil {
				return err
			}
			return a.writeRawOrTable(data, genericKVTable)
		},
	}
	if spec.date {
		cmd.Flags().StringVar(&date, "date", "", "date in YYYY-MM-DD (defaults to today)")
	}
	if spec.rangeFlags {
		cmd.Flags().StringVar(&from, "from", "", "start date in YYYY-MM-DD")
		cmd.Flags().StringVar(&to, "to", "", "end date in YYYY-MM-DD")
	}
	if spec.days {
		cmd.Flags().IntVar(&days, "days", spec.defaultDays, "number of days")
	}
	if spec.post {
		cmd.Flags().StringVar(&unit, "unit", "kg", "weight unit")
		cmd.Args = cobra.ExactArgs(1)
	}
	return cmd
}

func (a *app) getCommand(use, short, path string, table func([]byte) ([][]string, []string, error)) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, token, err := a.api()
			if err != nil {
				return err
			}
			data, err := client.GetJSON(token, path)
			if err != nil {
				return err
			}
			return a.writeRawOrTable(data, table)
		},
	}
}

func dateOrToday(date string) string {
	if date != "" {
		return date
	}
	return garmin.Today()
}

func weightGrams(weight float64, unit string) (int64, error) {
	switch strings.ToLower(unit) {
	case "kg":
		return int64(weight * 1000), nil
	case "lb", "lbs":
		return int64(weight * 453.592), nil
	default:
		return 0, errors.New("invalid unit; use kg or lbs")
	}
}

func applyDisplayName(path, displayName string) string {
	return strings.ReplaceAll(path, "{displayName}", url.PathEscape(displayName))
}
