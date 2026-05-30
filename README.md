# garmin-connect-cli

`garmin-connect-cli` is a Go command-line client for Garmin Connect.

It is designed for scripts, terminals, and AI agents: compact JSON is written to stdout by default, while prompts, errors, and progress messages go to stderr. The CLI does not maintain a local activity database, sync cache, SQLite store, Parquet files, DuckDB state, or background task queue.

## Status

This project uses Garmin Connect's private web/mobile APIs. Those APIs are unofficial and can change without notice.

Authentication and basic health data have been tested locally. Other commands follow Garmin endpoint behavior used by existing community clients, but may need adjustment if Garmin changes response shapes or routes.

## Install

Build from source:

```bash
go build -o garmin ./cmd/garmin
```

Optionally install into your Go bin directory:

```bash
go install ./cmd/garmin
```

## Authentication

Log in with your Garmin account:

```bash
garmin auth login --email user@example.com
```

The password is prompted interactively and is not passed through argv. If Garmin requires MFA, the CLI prompts for the code.

Check auth status:

```bash
garmin auth status --pretty
```

Remove stored tokens:

```bash
garmin auth logout
```

OAuth tokens are stored as JSON files under the XDG config directory with `0600` file permissions. Profiles are supported with `--profile` or `GARMIN_PROFILE`.

Examples:

```bash
garmin --profile personal auth status
GARMIN_PROFILE=work garmin profile show
```

## Output Contract

JSON is the stable scripting interface:

```bash
garmin health sleep
garmin health sleep --pretty
```

Human-readable tables are available where useful:

```bash
garmin activities list --format table
garmin devices list --format table
```

Downloads are explicit file operations:

```bash
garmin activities download 123456789 --type fit --output activity.fit.zip
garmin activities download 123456789 --type gpx --output -
```

## Commands

Authentication:

```bash
garmin auth login
garmin auth logout
garmin auth status
```

Profile:

```bash
garmin profile show
garmin profile settings
```

Activities:

```bash
garmin activities list --limit 20
garmin activities get ACTIVITY_ID
garmin activities download ACTIVITY_ID --type fit --output activity.fit.zip
garmin activities upload activity.fit
```

Devices:

```bash
garmin devices list
garmin devices get DEVICE_ID
```

Health:

```bash
garmin health summary
garmin health sleep
garmin health stress
garmin health heart-rate
garmin health body-battery
garmin health steps
garmin health calories
garmin health weight
garmin health weight-add 80.2 --unit kg
garmin health vo2max
garmin health training-readiness
garmin health training-status
garmin health hrv
garmin health fitness-age
garmin health lactate-threshold
garmin health race-predictions
garmin health endurance-score
garmin health hill-score
garmin health spo2
garmin health respiration
garmin health intensity-minutes
garmin health blood-pressure
garmin health hydration
garmin health personal-records
garmin health performance-summary
garmin health insights
```

There is intentionally no `sync` command and no `devices history` command.

## AI Agent Help

Use `help-json` for machine-readable command discovery:

```bash
garmin help-json
garmin help-json health sleep
garmin help-json activities download
```

This returns command paths, arguments, flags, defaults, required flags, global flags, examples, and unavailable commands.

## Development

Run tests:

```bash
go test ./...
```

Run vet:

```bash
go vet ./...
```

Build:

```bash
go build -o garmin ./cmd/garmin
```

## Design Goals

- JSON-first output for composition with tools like `jq`.
- No local Garmin data warehouse or sync subsystem.
- Explicit file output only for downloads.
- Minimal response modeling to avoid overfitting unstable private APIs.
- Profile-aware token persistence only.
- Help output that is usable by both humans and AI agents.
