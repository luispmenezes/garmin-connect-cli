package garmin

import (
	"fmt"
	"net/url"
	"time"
)

type DownloadEndpoint struct {
	Path      string
	Extension string
}

func ActivityListPath(start, limit int) string {
	return fmt.Sprintf("/activitylist-service/activities/search/activities?limit=%d&start=%d", limit, start)
}

func ActivityGetPath(id string) string {
	return "/activity-service/activity/" + url.PathEscape(id)
}

func ActivityDownloadEndpoint(id, typ string) (DownloadEndpoint, error) {
	switch typ {
	case "fit":
		return DownloadEndpoint{Path: "/download-service/files/activity/" + url.PathEscape(id), Extension: "fit.zip"}, nil
	case "gpx":
		return DownloadEndpoint{Path: "/download-service/export/gpx/activity/" + url.PathEscape(id), Extension: "gpx"}, nil
	case "tcx":
		return DownloadEndpoint{Path: "/download-service/export/tcx/activity/" + url.PathEscape(id), Extension: "tcx"}, nil
	case "kml":
		return DownloadEndpoint{Path: "/download-service/export/kml/activity/" + url.PathEscape(id), Extension: "kml"}, nil
	default:
		return DownloadEndpoint{}, fmt.Errorf("unknown download type %q; supported: fit, gpx, tcx, kml", typ)
	}
}

func DeviceListPath() string { return "/device-service/deviceregistration/devices" }

func DeviceGetPath(id string) string {
	return "/device-service/deviceservice/device-info/settings/" + url.PathEscape(id)
}

func ProfilePath() string { return "/userprofile-service/socialProfile" }

func SettingsPath() string { return "/userprofile-service/userprofile/user-settings" }

func WorkoutListPath() string { return "/workout-service/workouts" }

func WorkoutGetPath(id string) string {
	return "/workout-service/workout/" + url.PathEscape(id) + "?includeAudioNotes=false"
}

func WorkoutCreatePath() string { return "/workout-service/workout" }

func WorkoutUpdatePath(id string) string {
	return "/workout-service/workout/" + url.PathEscape(id)
}

func WorkoutDeletePath(id string) string {
	return "/workout-service/workout/" + url.PathEscape(id)
}

func WorkoutSchedulePath(workoutID string) string {
	return "/workout-service/schedule/" + url.PathEscape(workoutID)
}

func WorkoutScheduleGetPath(scheduleID string) string {
	return "/workout-service/schedule/" + url.PathEscape(scheduleID) + "?includeAudioNotes=false"
}

func WorkoutUnschedulePath(scheduleID string) string {
	return "/workout-service/schedule/" + url.PathEscape(scheduleID)
}

func CalendarWorkoutSummaryPath(startDate string) string {
	return "/calendar-service/workout/schedule/summary/?startDate=" + url.QueryEscape(startDate)
}

func Today() string { return time.Now().Format("2006-01-02") }

func AddDays(date string, days int) (string, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", err
	}
	return t.AddDate(0, 0, days).Format("2006-01-02"), nil
}
