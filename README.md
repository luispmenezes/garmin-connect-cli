# garmin-connect-cli

A small Go CLI for Garmin Connect.

It logs in to your Garmin account and prints profile, activity, device, and health data as JSON by default, making it easy to use with tools like `jq` or shell scripts.

## Install

```bash
go install github.com/luispmenezes/garmin-connect-cli/cmd/garmin@latest
```

Or build locally:

```bash
go build -o garmin ./cmd/garmin
```

## Login

```bash
garmin auth login --email user@example.com
```

The CLI prompts for your password and MFA code when needed. Tokens are stored in your user config directory and can be removed with:

```bash
garmin auth logout
```

Check login status:

```bash
garmin auth status --pretty
```

## Examples

Get last night's sleep:

```bash
garmin health sleep --pretty
```

List recent activities:

```bash
garmin activities list --limit 20
```

Show activities as a table:

```bash
garmin activities list --format table
```

Download an activity:

```bash
garmin activities download ACTIVITY_ID --type fit --output activity.fit.zip
```

Show devices:

```bash
garmin devices list --format table
```

List workouts and scheduled workout summaries:

```bash
garmin workouts list --pretty
garmin calendar workouts --start 2026-06-01 --pretty
```

Create a workout from Garmin JSON:

```bash
garmin workouts create examples/workout-running-minimal.json
```

The example workout is intentionally a raw Garmin payload for manual testing. Garmin may reject incomplete segment structures, and the CLI sends the file contents unchanged.

Use a separate profile:

```bash
garmin --profile personal health sleep
```

## Commands

```bash
garmin auth login|logout|status
garmin profile show|settings
garmin activities list|get|download|upload
garmin devices list|get
garmin health summary|sleep|stress|heart-rate|body-battery|steps|calories|weight|weight-add
garmin health vo2max|training-readiness|training-status|hrv|fitness-age
garmin health lactate-threshold|race-predictions|endurance-score|hill-score
garmin health spo2|respiration|intensity-minutes|blood-pressure|hydration
garmin health personal-records|performance-summary|insights
garmin workouts list|get|create|update|delete|schedule|schedule-get|unschedule
garmin calendar workouts
```

Run normal help:

```bash
garmin --help
garmin health sleep --help
```

Machine-readable help is also available:

```bash
garmin help-json
garmin help-json health sleep
```

## Output

The default output is compact JSON:

```bash
garmin health sleep
```

Use `--pretty` for indented JSON:

```bash
garmin health sleep --pretty
```

Use `--format table` where a human summary is useful:

```bash
garmin devices list --format table
```

## Development

```bash
go test ./...
go vet ./...
go build -o garmin ./cmd/garmin
```

## Note

Garmin Connect does not provide an official public API for this use case. This CLI uses Garmin's private web/mobile endpoints, so commands may need updates if Garmin changes them.
