package garmin

import "testing"

func TestActivityDownloadEndpoint(t *testing.T) {
	got, err := ActivityDownloadEndpoint("123", "fit")
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "/download-service/files/activity/123" || got.Extension != "fit.zip" {
		t.Fatalf("unexpected endpoint: %#v", got)
	}
	if _, err := ActivityDownloadEndpoint("123", "csv"); err == nil {
		t.Fatal("expected unsupported type error")
	}
}

func TestActivityEndpoints(t *testing.T) {
	if got := ActivitySplitsPath("activity 1/2"); got != "/activity-service/activity/activity%201%2F2/splits" {
		t.Fatalf("splits path = %q", got)
	}
	if got := ActivityStatsPath("activity 1/2"); got != "/activity-service/activity/activity%201%2F2/split_summaries" {
		t.Fatalf("stats path = %q", got)
	}
}

func TestAddDays(t *testing.T) {
	got, err := AddDays("2026-05-30", -6)
	if err != nil {
		t.Fatal(err)
	}
	if got != "2026-05-24" {
		t.Fatalf("got %s", got)
	}
}

func TestWorkoutEndpoints(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"list", WorkoutListPath(), "/workout-service/workouts"},
		{"get", WorkoutGetPath("workout 1/2"), "/workout-service/workout/workout%201%2F2?includeAudioNotes=false"},
		{"create", WorkoutCreatePath(), "/workout-service/workout"},
		{"update", WorkoutUpdatePath("workout 1/2"), "/workout-service/workout/workout%201%2F2"},
		{"delete", WorkoutDeletePath("workout 1/2"), "/workout-service/workout/workout%201%2F2"},
		{"schedule", WorkoutSchedulePath("workout 1/2"), "/workout-service/schedule/workout%201%2F2"},
		{"schedule get", WorkoutScheduleGetPath("schedule 1/2"), "/workout-service/schedule/schedule%201%2F2?includeAudioNotes=false"},
		{"unschedule", WorkoutUnschedulePath("schedule 1/2"), "/workout-service/schedule/schedule%201%2F2"},
		{"calendar", CalendarWorkoutSummaryPath("2026-06-01"), "/calendar-service/workout/schedule/summary/?startDate=2026-06-01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("path = %q, want %q", tt.got, tt.want)
			}
		})
	}
}
