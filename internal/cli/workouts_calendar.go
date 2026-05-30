package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/luispmenezes/garmin-connect-cli/internal/garmin"
	"github.com/spf13/cobra"
)

func (a *app) workoutsCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "workouts", Short: "Manage Garmin workouts"}
	cmd.AddCommand(a.workoutsListCommand())
	cmd.AddCommand(a.workoutsGetCommand())
	cmd.AddCommand(a.workoutsCreateCommand())
	cmd.AddCommand(a.workoutsUpdateCommand())
	cmd.AddCommand(a.workoutsDeleteCommand())
	cmd.AddCommand(a.workoutsScheduleCommand())
	cmd.AddCommand(a.workoutsScheduleGetCommand())
	cmd.AddCommand(a.workoutsUnscheduleCommand())
	return cmd
}

func (a *app) workoutsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List workouts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, token, err := a.api()
			if err != nil {
				return err
			}
			data, err := client.GetJSON(token, garmin.WorkoutListPath())
			if err != nil {
				return err
			}
			return a.out().WriteJSON(data)
		},
	}
}

func (a *app) workoutsGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get WORKOUT_ID",
		Short: "Show workout details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, token, err := a.api()
			if err != nil {
				return err
			}
			data, err := client.GetJSON(token, garmin.WorkoutGetPath(args[0]))
			if err != nil {
				return err
			}
			return a.out().WriteJSON(data)
		},
	}
}

func (a *app) workoutsCreateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "create workout.json",
		Short: "Create a workout from raw Garmin JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readJSONInput(args[0])
			if err != nil {
				return err
			}
			client, token, err := a.api()
			if err != nil {
				return err
			}
			data, err := client.PostRawJSON(token, garmin.WorkoutCreatePath(), body)
			if err != nil {
				return err
			}
			return a.out().WriteJSON(data)
		},
	}
}

func (a *app) workoutsUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update WORKOUT_ID workout.json",
		Short: "Update a workout from raw Garmin JSON",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			body, err := readJSONInput(args[1])
			if err != nil {
				return err
			}
			client, token, err := a.api()
			if err != nil {
				return err
			}
			data, err := client.PutRawJSON(token, garmin.WorkoutUpdatePath(args[0]), body)
			if err != nil {
				return err
			}
			return a.out().WriteJSON(data)
		},
	}
}

func (a *app) workoutsDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete WORKOUT_ID",
		Short: "Delete a workout",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, token, err := a.api()
			if err != nil {
				return err
			}
			data, err := client.DeleteJSON(token, garmin.WorkoutDeletePath(args[0]))
			if err != nil {
				return err
			}
			return a.writeDeleteResult(data, "deleted", args[0])
		},
	}
}

func (a *app) workoutsScheduleCommand() *cobra.Command {
	var date string
	cmd := &cobra.Command{
		Use:   "schedule WORKOUT_ID",
		Short: "Schedule a workout",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateDate(date); err != nil {
				return err
			}
			client, token, err := a.api()
			if err != nil {
				return err
			}
			data, err := client.PostJSON(token, garmin.WorkoutSchedulePath(args[0]), date)
			if err != nil {
				return err
			}
			return a.out().WriteJSON(data)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "schedule date in YYYY-MM-DD")
	_ = cmd.MarkFlagRequired("date")
	return cmd
}

func (a *app) workoutsScheduleGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "schedule-get SCHEDULE_ID",
		Short: "Show scheduled workout details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, token, err := a.api()
			if err != nil {
				return err
			}
			data, err := client.GetJSON(token, garmin.WorkoutScheduleGetPath(args[0]))
			if err != nil {
				return err
			}
			return a.out().WriteJSON(data)
		},
	}
}

func (a *app) workoutsUnscheduleCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unschedule SCHEDULE_ID",
		Short: "Remove a workout from the calendar",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, token, err := a.api()
			if err != nil {
				return err
			}
			data, err := client.DeleteJSON(token, garmin.WorkoutUnschedulePath(args[0]))
			if err != nil {
				return err
			}
			return a.writeDeleteResult(data, "unscheduled", args[0])
		},
	}
}

func (a *app) calendarCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "calendar", Short: "Show Garmin calendar data"}
	var start string
	workouts := &cobra.Command{
		Use:   "workouts",
		Short: "Show scheduled workout summaries",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateDate(start); err != nil {
				return err
			}
			client, token, err := a.api()
			if err != nil {
				return err
			}
			data, err := client.GetJSON(token, garmin.CalendarWorkoutSummaryPath(start))
			if err != nil {
				return err
			}
			return a.out().WriteJSON(data)
		},
	}
	workouts.Flags().StringVar(&start, "start", "", "start date in YYYY-MM-DD")
	_ = workouts.MarkFlagRequired("start")
	cmd.AddCommand(workouts)
	return cmd
}

func (a *app) writeDeleteResult(data []byte, field, id string) error {
	if len(bytes.TrimSpace(data)) > 0 {
		return a.out().WriteJSON(data)
	}
	return a.out().WriteValue(map[string]any{field: true, "id": id})
}

func readJSONInput(path string) ([]byte, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, errors.New("JSON input is empty")
	}
	return data, nil
}

func validateDate(date string) error {
	if date == "" {
		return errors.New("date is required in YYYY-MM-DD format")
	}
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil || parsed.Format("2006-01-02") != date {
		return fmt.Errorf("invalid date %q; use YYYY-MM-DD", date)
	}
	return nil
}
