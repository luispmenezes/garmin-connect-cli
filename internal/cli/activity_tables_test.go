package cli

import "testing"

// Payload shapes confirmed against the live Garmin API:
//   /activity-service/activity/{id}/splits         -> {"lapDTOs":[...]}
//   /activity-service/activity/{id}/split_summaries -> {"splitSummaries":[...]}

func TestActivitySplitsTable(t *testing.T) {
	data := []byte(`{"activityId":1,"lapDTOs":[
		{"lapIndex":1,"distance":1000.0,"duration":314.0,"averageHR":132,"elevationGain":4.0},
		{"lapIndex":2,"distance":340.0,"duration":106.0,"averageHR":null}
	]}`)
	rows, headers, err := activitySplitsTable(data, pacePerKM)
	if err != nil {
		t.Fatal(err)
	}
	if len(headers) != 6 || headers[3] != "Pace" {
		t.Fatalf("headers = %v", headers)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if rows[0][3] != "5:14 /km" { // 1000m in 314s
		t.Fatalf("pace = %q", rows[0][3])
	}
	if rows[0][5] != "4 m" {
		t.Fatalf("ascent = %q", rows[0][5])
	}
	if rows[1][4] != "-" { // null HR
		t.Fatalf("missing HR = %q", rows[1][4])
	}
}

func TestActivityStatsTable(t *testing.T) {
	data := []byte(`{"activityId":1,"splitSummaries":[
		{"splitType":"RWD_RUN","noOfSplits":1,"distance":7335.46,"duration":2235.354,"averageHR":142,"elevationGain":26.09}
	]}`)
	rows, headers, err := activityStatsTable(data, pacePerKM)
	if err != nil {
		t.Fatal(err)
	}
	if len(headers) != 7 || headers[0] != "Type" {
		t.Fatalf("headers = %v", headers)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if rows[0][0] != "RWD_RUN" || rows[0][6] != "26 m" {
		t.Fatalf("row = %v", rows[0])
	}
}

func TestActivityStatsTableEmpty(t *testing.T) {
	// Garmin returns an empty splitSummaries array for some activities (e.g. cycling).
	rows, _, err := activityStatsTable([]byte(`{"activityId":1,"splitSummaries":[]}`), pacePerKM)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("want 0 rows, got %d", len(rows))
	}
}

func TestSpeedUnitForType(t *testing.T) {
	tests := map[string]speedUnit{
		"running":             pacePerKM,
		"trail_running":       pacePerKM,
		"walking":             pacePerKM,
		"hiking":              pacePerKM,
		"road_biking":         speedKMH,
		"cycling":             speedKMH,
		"mountain_biking":     speedKMH,
		"indoor_cycling":      speedKMH,
		"lap_swimming":        pacePer100m,
		"open_water_swimming": pacePer100m,
		"":                    pacePerKM,
	}
	for typeKey, want := range tests {
		if got := speedUnitForType(typeKey); got != want {
			t.Errorf("speedUnitForType(%q) = %d, want %d", typeKey, got, want)
		}
	}
}

func TestPaceOrSpeed(t *testing.T) {
	// 1000m in 314s
	if got := paceOrSpeed(1000.0, 314.0, pacePerKM); got != "5:14 /km" {
		t.Errorf("pace = %q", got)
	}
	// 141050m in 26805s -> ~18.9 km/h
	if got := paceOrSpeed(141050.0, 26805.0, speedKMH); got != "18.9 km/h" {
		t.Errorf("speed = %q", got)
	}
	// 100m in 95s
	if got := paceOrSpeed(100.0, 95.0, pacePer100m); got != "1:35 /100m" {
		t.Errorf("swim pace = %q", got)
	}
	if got := paceOrSpeed(0.0, 100.0, speedKMH); got != "-" {
		t.Errorf("zero distance = %q", got)
	}
}

func TestSpeedUnitFromActivity(t *testing.T) {
	if got := speedUnitFromActivity([]byte(`{"activityTypeDTO":{"typeKey":"road_biking"}}`)); got != speedKMH {
		t.Errorf("got %d, want speedKMH", got)
	}
	if got := speedUnitFromActivity([]byte(`{"activityTypeDTO":{"typeKey":"running"}}`)); got != pacePerKM {
		t.Errorf("got %d, want pacePerKM", got)
	}
	if got := speedUnitFromActivity([]byte(`garbage`)); got != pacePerKM {
		t.Errorf("malformed payload should fall back to pacePerKM, got %d", got)
	}
}
