package cli

import (
	"encoding/json"
	"testing"
)

func TestCourseSavePayload(t *testing.T) {
	// The parse step returns geoPoints with null elevation and null derived
	// metadata; courseSavePayload must compute the fields the save endpoint
	// requires (startPoint, boundingBox, distance) and tag the course.
	parsed := []byte(`{
		"courseName": "Test Loop",
		"distanceMeter": null,
		"elevationGainMeter": null,
		"geoPoints": [
			{"latitude": 40.6, "longitude": -8.6, "elevation": null, "distance": 0},
			{"latitude": 40.9, "longitude": -8.2, "elevation": null, "distance": 5000}
		]
	}`)

	out, err := courseSavePayload(parsed, 10, 2)
	if err != nil {
		t.Fatalf("courseSavePayload: %v", err)
	}
	var d map[string]any
	if err := json.Unmarshal(out, &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if d["activityTypePk"].(float64) != 10 {
		t.Errorf("activityTypePk = %v, want 10", d["activityTypePk"])
	}
	if d["rulePK"].(float64) != 2 {
		t.Errorf("rulePK = %v, want 2", d["rulePK"])
	}
	if d["sourceTypeId"].(float64) != 3 {
		t.Errorf("sourceTypeId = %v, want 3", d["sourceTypeId"])
	}
	if d["coordinateSystem"] != "WGS84" {
		t.Errorf("coordinateSystem = %v, want WGS84", d["coordinateSystem"])
	}
	if d["distanceMeter"].(float64) != 5000 {
		t.Errorf("distanceMeter = %v, want 5000 (last point distance)", d["distanceMeter"])
	}

	start := d["startPoint"].(map[string]any)
	if start["latitude"].(float64) != 40.6 || start["longitude"].(float64) != -8.6 {
		t.Errorf("startPoint = %v, want first geoPoint", start)
	}

	bb := d["boundingBox"].(map[string]any)
	ll := bb["lowerLeft"].(map[string]any)
	ur := bb["upperRight"].(map[string]any)
	if ll["latitude"].(float64) != 40.6 || ll["longitude"].(float64) != -8.6 {
		t.Errorf("boundingBox lowerLeft = %v", ll)
	}
	if ur["latitude"].(float64) != 40.9 || ur["longitude"].(float64) != -8.2 {
		t.Errorf("boundingBox upperRight = %v", ur)
	}

	// Null elevations must be coerced to a number so the float field validates.
	pts := d["geoPoints"].([]any)
	if pts[0].(map[string]any)["elevation"].(float64) != 0 {
		t.Errorf("null elevation not coerced to 0")
	}
}

func TestCourseSavePayloadRejectsTooFewPoints(t *testing.T) {
	if _, err := courseSavePayload([]byte(`{"geoPoints":[{"latitude":1,"longitude":2}]}`), 10, 2); err == nil {
		t.Fatal("expected error for fewer than two points")
	}
}
