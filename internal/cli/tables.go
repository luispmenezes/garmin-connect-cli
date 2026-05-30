package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

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
