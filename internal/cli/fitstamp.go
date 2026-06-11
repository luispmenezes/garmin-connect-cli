package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/luispmenezes/garmin-connect-cli/internal/fit"
)

// deviceFitStamp finds the device matching id (its deviceId or unitId) in the
// devices-list JSON and maps it to a FIT file_id stamp: Garmin as manufacturer,
// the product code parsed from the part number (006-B3290-00 -> 3290), and the
// serial number taken from the numeric unit id. It returns the stamp and the
// device's display name.
func deviceFitStamp(devicesJSON []byte, id string) (fit.Stamp, string, error) {
	var devices []map[string]any
	if err := json.Unmarshal(devicesJSON, &devices); err != nil {
		return fit.Stamp{}, "", fmt.Errorf("parse devices: %w", err)
	}
	for _, d := range devices {
		if valueString(d, "deviceId") != id && valueString(d, "unitId") != id {
			continue
		}
		name := firstString(d, "productDisplayName", "displayName")
		product, err := productFromPartNumber(valueString(d, "partNumber"))
		if err != nil {
			return fit.Stamp{}, name, err
		}
		serial, err := strconv.ParseUint(valueString(d, "unitId"), 10, 32)
		if err != nil {
			return fit.Stamp{}, name, fmt.Errorf("device %s has no numeric unitId for the FIT serial number", id)
		}
		mfg := uint16(1) // Garmin
		ser := uint32(serial)
		return fit.Stamp{Manufacturer: &mfg, Product: &product, Serial: &ser}, name, nil
	}
	return fit.Stamp{}, "", fmt.Errorf("no registered device with id %s", id)
}

// productFromPartNumber extracts the FIT product code from a Garmin part number
// such as "006-B3290-00" -> 3290.
func productFromPartNumber(part string) (uint16, error) {
	fields := strings.Split(part, "-")
	if len(fields) < 2 {
		return 0, fmt.Errorf("unrecognized part number %q", part)
	}
	n, err := strconv.ParseUint(strings.TrimPrefix(fields[1], "B"), 10, 16)
	if err != nil {
		return 0, fmt.Errorf("cannot derive product code from part number %q", part)
	}
	return uint16(n), nil
}
