package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/luispmenezes/garmin-connect-cli/internal/output"
)

func activitiesTable(data []byte) ([][]string, []string, error) {
	var activities []map[string]any
	if err := json.Unmarshal(data, &activities); err != nil {
		return nil, nil, err
	}
	rows := make([][]string, 0, len(activities))
	for _, a := range activities {
		rows = append(rows, []string{
			valueString(a, "activityId"),
			shortDate(valueString(a, "startTimeLocal")),
			nestedString(a, "activityType", "typeKey"),
			distanceKM(a["distance"]),
			duration(a["duration"]),
			valueString(a, "averageHR"),
		})
	}
	return rows, []string{"ID", "Date", "Type", "Distance", "Duration", "HR"}, nil
}

func activitySplitsTable(data []byte, unit speedUnit) ([][]string, []string, error) {
	var obj struct {
		LapDTOs []map[string]any `json:"lapDTOs"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, nil, err
	}
	rows := make([][]string, 0, len(obj.LapDTOs))
	for _, l := range obj.LapDTOs {
		rows = append(rows, []string{
			valueString(l, "lapIndex"),
			distanceKM(l["distance"]),
			duration(l["duration"]),
			paceOrSpeed(l["distance"], l["duration"], unit),
			valueString(l, "averageHR"),
			elevation(l["elevationGain"]),
		})
	}
	return rows, []string{"Lap", "Distance", "Duration", speedHeader(unit), "HR", "Ascent"}, nil
}

func activityStatsTable(data []byte, unit speedUnit) ([][]string, []string, error) {
	var obj struct {
		SplitSummaries []map[string]any `json:"splitSummaries"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, nil, err
	}
	rows := make([][]string, 0, len(obj.SplitSummaries))
	for _, s := range obj.SplitSummaries {
		rows = append(rows, []string{
			valueString(s, "splitType"),
			valueString(s, "noOfSplits"),
			distanceKM(s["distance"]),
			duration(s["duration"]),
			paceOrSpeed(s["distance"], s["duration"], unit),
			valueString(s, "averageHR"),
			elevation(s["elevationGain"]),
		})
	}
	return rows, []string{"Type", "Splits", "Distance", "Duration", speedHeader(unit), "HR", "Ascent"}, nil
}

func firstVal(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return v
		}
	}
	return nil
}

func coursesTable(data []byte) ([][]string, []string, error) {
	// The course list endpoint may return a bare array or wrap it under a key
	// such as "coursesForUser"/"courses"; handle both.
	var courses []map[string]any
	if err := json.Unmarshal(data, &courses); err != nil {
		var wrapper map[string]json.RawMessage
		if err2 := json.Unmarshal(data, &wrapper); err2 != nil {
			return nil, nil, err
		}
		for _, key := range []string{"coursesForUser", "courses", "items"} {
			if raw, ok := wrapper[key]; ok {
				if err2 := json.Unmarshal(raw, &courses); err2 == nil {
					break
				}
			}
		}
	}
	rows := make([][]string, 0, len(courses))
	for _, c := range courses {
		rows = append(rows, []string{
			firstString(c, "courseId", "id"),
			output.Truncate(firstString(c, "courseName", "name"), 32),
			nestedString(c, "activityType", "typeKey"),
			distanceKM(firstVal(c, "distanceInMeters", "distanceMeter", "distance")),
			elevation(firstVal(c, "elevationGainInMeters", "elevationGainMeter", "elevationGain")),
		})
	}
	return rows, []string{"ID", "Name", "Type", "Distance", "Ascent"}, nil
}

func devicesTable(data []byte) ([][]string, []string, error) {
	var devices []map[string]any
	if err := json.Unmarshal(data, &devices); err != nil {
		return nil, nil, err
	}
	rows := make([][]string, 0, len(devices))
	for _, d := range devices {
		name := firstString(d, "displayName", "deviceTypeName", "friendlyName")
		rows = append(rows, []string{
			output.Truncate(name, 24),
			output.Truncate(valueString(d, "partNumber"), 24),
			output.Truncate(valueString(d, "currentFirmwareVersion"), 16),
			shortDate(valueString(d, "lastSyncTime")),
		})
	}
	return rows, []string{"Device", "Model", "Software", "Last Sync"}, nil
}

func profileTable(data []byte) ([][]string, []string, error) {
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, nil, err
	}
	rows := [][]string{
		{"displayName", firstString(obj, "displayName", "fullName", "userName")},
		{"userName", valueString(obj, "userName")},
		{"profileId", valueString(obj, "profileId")},
	}
	return rows, []string{"Field", "Value"}, nil
}

func genericKVTable(data []byte) ([][]string, []string, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, nil, err
	}
	rows := make([][]string, 0)
	switch x := v.(type) {
	case map[string]any:
		i := 0
		for k, val := range x {
			rows = append(rows, []string{k, scalar(val)})
			i++
			if i >= 30 {
				break
			}
		}
	case []any:
		for i, val := range x {
			rows = append(rows, []string{strconv.Itoa(i), scalar(val)})
			if i >= 29 {
				break
			}
		}
	default:
		rows = append(rows, []string{"value", scalar(x)})
	}
	return rows, []string{"Field", "Value"}, nil
}

func valueString(m map[string]any, key string) string {
	return scalar(m[key])
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if s := scalar(m[key]); s != "" && s != "-" {
			return s
		}
	}
	return "-"
}

func nestedString(m map[string]any, key, nested string) string {
	child, _ := m[key].(map[string]any)
	return scalar(child[nested])
}

func scalar(v any) string {
	switch x := v.(type) {
	case nil:
		return "-"
	case string:
		if x == "" {
			return "-"
		}
		return x
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return fmt.Sprintf("%.2f", x)
	case bool:
		return strconv.FormatBool(x)
	default:
		data, _ := json.Marshal(x)
		if len(data) > 80 {
			return string(data[:80]) + "..."
		}
		return string(data)
	}
}

func shortDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	if s == "" {
		return "-"
	}
	return s
}

func distanceKM(v any) string {
	f, ok := v.(float64)
	if !ok {
		return "-"
	}
	return fmt.Sprintf("%.2f km", f/1000)
}

func duration(v any) string {
	f, ok := v.(float64)
	if !ok {
		return "-"
	}
	sec := int(f)
	return fmt.Sprintf("%d:%02d:%02d", sec/3600, (sec%3600)/60, sec%60)
}

// speedUnit selects how distance-over-time is rendered, matching the Garmin
// app's per-sport convention: pace (min/km) for foot sports, speed (km/h) for
// cycling, and pace per 100m for swimming.
type speedUnit int

const (
	pacePerKM speedUnit = iota
	speedKMH
	pacePer100m
)

// speedUnitForType maps a Garmin activity typeKey (e.g. "running",
// "road_biking", "lap_swimming") to its display unit. typeKeys have many
// variants, so it matches on substrings rather than an exhaustive list.
func speedUnitForType(typeKey string) speedUnit {
	k := strings.ToLower(typeKey)
	switch {
	case strings.Contains(k, "cycling"), strings.Contains(k, "biking"), strings.Contains(k, "bike"):
		return speedKMH
	case strings.Contains(k, "swim"):
		return pacePer100m
	default:
		return pacePerKM
	}
}

// speedUnitFromActivity derives the display unit from an activity-detail
// payload (/activity-service/activity/{id}). On any parse failure it falls back
// to pace per km.
func speedUnitFromActivity(data []byte) speedUnit {
	var obj struct {
		ActivityTypeDTO struct {
			TypeKey string `json:"typeKey"`
		} `json:"activityTypeDTO"`
	}
	_ = json.Unmarshal(data, &obj)
	return speedUnitForType(obj.ActivityTypeDTO.TypeKey)
}

func speedHeader(unit speedUnit) string {
	if unit == speedKMH {
		return "Speed"
	}
	return "Pace"
}

func paceOrSpeed(distance, dur any, unit speedUnit) string {
	d, ok := distance.(float64)
	if !ok || d <= 0 {
		return "-"
	}
	t, ok := dur.(float64)
	if !ok || t <= 0 {
		return "-"
	}
	switch unit {
	case speedKMH:
		return fmt.Sprintf("%.1f km/h", (d/1000)/(t/3600))
	case pacePer100m:
		sec := int(t / (d / 100))
		return fmt.Sprintf("%d:%02d /100m", sec/60, sec%60)
	default:
		sec := int(t / (d / 1000))
		return fmt.Sprintf("%d:%02d /km", sec/60, sec%60)
	}
}

func elevation(v any) string {
	f, ok := v.(float64)
	if !ok {
		return "-"
	}
	return fmt.Sprintf("%.0f m", f)
}
