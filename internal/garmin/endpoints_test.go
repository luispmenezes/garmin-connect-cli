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

func TestAddDays(t *testing.T) {
	got, err := AddDays("2026-05-30", -6)
	if err != nil {
		t.Fatal(err)
	}
	if got != "2026-05-24" {
		t.Fatalf("got %s", got)
	}
}
