package cli

import "testing"

const devicesJSON = `[
  {"deviceId":3426524463,"unitId":3426524463,"productDisplayName":"Edge 1030 Plus","partNumber":"006-B3570-00","serialNumber":"6ET264047"},
  {"deviceId":3309787599,"unitId":3309787599,"productDisplayName":"fenix 6 Pro","partNumber":"006-B3290-00","serialNumber":"63N062568"}
]`

func TestDeviceFitStamp(t *testing.T) {
	stamp, name, err := deviceFitStamp([]byte(devicesJSON), "3309787599")
	if err != nil {
		t.Fatal(err)
	}
	if name != "fenix 6 Pro" {
		t.Errorf("name = %q", name)
	}
	if stamp.Manufacturer == nil || *stamp.Manufacturer != 1 {
		t.Errorf("manufacturer = %v, want 1", stamp.Manufacturer)
	}
	if stamp.Product == nil || *stamp.Product != 3290 {
		t.Errorf("product = %v, want 3290", stamp.Product)
	}
	if stamp.Serial == nil || *stamp.Serial != 3309787599 {
		t.Errorf("serial = %v, want 3309787599", stamp.Serial)
	}
}

func TestDeviceFitStampNotFound(t *testing.T) {
	if _, _, err := deviceFitStamp([]byte(devicesJSON), "123"); err == nil {
		t.Fatal("expected error for unknown device")
	}
}

func TestProductFromPartNumber(t *testing.T) {
	cases := map[string]uint16{
		"006-B3290-00": 3290,
		"006-B3570-00": 3570,
	}
	for part, want := range cases {
		got, err := productFromPartNumber(part)
		if err != nil {
			t.Errorf("%s: %v", part, err)
			continue
		}
		if got != want {
			t.Errorf("%s: got %d want %d", part, got, want)
		}
	}
	if _, err := productFromPartNumber("garbage"); err == nil {
		t.Error("expected error for malformed part number")
	}
}
