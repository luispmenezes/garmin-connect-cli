// Package fit provides a minimal, surgical patcher for a FIT file's file_id
// message. It exists so the CLI can stamp an activity FIT with a chosen
// device's identity (manufacturer/product/serial_number) before upload, which
// is how Garmin Connect attributes an activity to a device — attribution comes
// from the FIT file_id, not from the upload request.
//
// The patcher deliberately does NOT decode the whole file. The FIT protocol
// requires file_id to be the first record, and the fields we touch are
// fixed-width, so the edit is size-preserving and only the trailing CRC needs
// recomputing.
package fit

import (
	"encoding/binary"
	"fmt"
)

// Stamp holds the file_id values to overwrite. A nil field is left unchanged.
type Stamp struct {
	Manufacturer *uint16
	Product      *uint16
	Serial       *uint32
}

// IsZero reports whether the stamp would change nothing.
func (s Stamp) IsZero() bool {
	return s.Manufacturer == nil && s.Product == nil && s.Serial == nil
}

// file_id global message number and the field definition numbers we patch.
const (
	fileIDGlobalMsgNum = 0
	fieldManufacturer  = 1
	fieldProduct       = 2
	fieldSerialNumber  = 3
)

// crcTable is the standard FIT CRC-16 nibble lookup table.
var crcTable = [16]uint16{
	0x0000, 0xCC01, 0xD801, 0x1400, 0xF001, 0x3C00, 0x2800, 0xE401,
	0xA001, 0x6C00, 0x7800, 0xB401, 0x5000, 0x9C01, 0x8801, 0x4400,
}

// crc16 computes the FIT CRC-16 over buf.
func crc16(buf []byte) uint16 {
	var crc uint16
	for _, b := range buf {
		for _, nibble := range [2]uint16{uint16(b) & 0x0F, uint16(b>>4) & 0x0F} {
			tmp := crcTable[crc&0x0F]
			crc = (crc >> 4) & 0x0FFF
			crc = crc ^ tmp ^ crcTable[nibble]
		}
	}
	return crc
}

// StampFile returns a copy of data with the file_id fields named in s
// overwritten and the trailing CRC recomputed. The input slice is not modified.
//
// It errors rather than risk corrupting the file when the input is not a
// recognizable FIT whose first record is a file_id definition, or when a
// requested field is absent from the file_id (which would require resizing).
func StampFile(data []byte, s Stamp) ([]byte, error) {
	if s.IsZero() {
		out := make([]byte, len(data))
		copy(out, data)
		return out, nil
	}
	if len(data) < 14 {
		return nil, fmt.Errorf("not a FIT file: too short (%d bytes)", len(data))
	}

	headerSize := int(data[0])
	if headerSize != 12 && headerSize != 14 {
		return nil, fmt.Errorf("unexpected FIT header size %d (want 12 or 14)", headerSize)
	}
	if string(data[8:12]) != ".FIT" {
		return nil, fmt.Errorf("not a FIT file: missing .FIT signature")
	}
	if len(data) < headerSize+2 {
		return nil, fmt.Errorf("FIT file truncated")
	}

	out := make([]byte, len(data))
	copy(out, data)

	// The first record must be the file_id definition message.
	pos := headerSize
	recHeader := out[pos]
	if recHeader&0x80 != 0 || recHeader&0x40 == 0 {
		return nil, fmt.Errorf("first FIT record is not a definition message (header 0x%02x)", recHeader)
	}
	hasDevFields := recHeader&0x20 != 0
	pos++ // record header

	// Definition: reserved(1), architecture(1), global msg num(2), num fields(1).
	if pos+5 > len(out) {
		return nil, fmt.Errorf("FIT definition truncated")
	}
	pos++ // reserved
	bigEndian := out[pos] != 0
	pos++ // architecture
	order := binary.ByteOrder(binary.LittleEndian)
	if bigEndian {
		order = binary.BigEndian
	}
	globalMsgNum := order.Uint16(out[pos : pos+2])
	pos += 2
	if globalMsgNum != fileIDGlobalMsgNum {
		return nil, fmt.Errorf("first FIT record is global message %d, not file_id (0)", globalMsgNum)
	}
	numFields := int(out[pos])
	pos++

	// Walk the field definitions, recording each field's offset within the data
	// message. Field bytes are laid out in definition order.
	type fieldLoc struct{ offset, size int }
	locs := make(map[int]fieldLoc, numFields)
	dataOffset := 0
	for i := 0; i < numFields; i++ {
		if pos+3 > len(out) {
			return nil, fmt.Errorf("FIT field definition truncated")
		}
		fieldNum := int(out[pos])
		size := int(out[pos+1])
		pos += 3 // field def num, size, base type
		locs[fieldNum] = fieldLoc{offset: dataOffset, size: size}
		dataOffset += size
	}
	if hasDevFields {
		// Skip developer field definitions; they sit after the standard fields
		// in the data message, so they don't affect the offsets we patch.
		if pos >= len(out) {
			return nil, fmt.Errorf("FIT developer field count truncated")
		}
		numDevFields := int(out[pos])
		pos++
		pos += numDevFields * 3
	}

	// The data message immediately follows the definition.
	if pos >= len(out) {
		return nil, fmt.Errorf("FIT file_id data message missing")
	}
	dataRecHeader := out[pos]
	if dataRecHeader&0x80 != 0 || dataRecHeader&0x40 != 0 {
		return nil, fmt.Errorf("expected file_id data message, got record header 0x%02x", dataRecHeader)
	}
	dataStart := pos + 1
	if dataStart+dataOffset > len(out) {
		return nil, fmt.Errorf("FIT file_id data message truncated")
	}

	patch := func(fieldNum, wantSize int, write func(b []byte)) error {
		loc, ok := locs[fieldNum]
		if !ok {
			return fmt.Errorf("file_id has no field %d to set", fieldNum)
		}
		if loc.size != wantSize {
			return fmt.Errorf("file_id field %d has unexpected size %d (want %d)", fieldNum, loc.size, wantSize)
		}
		write(out[dataStart+loc.offset : dataStart+loc.offset+wantSize])
		return nil
	}

	if s.Manufacturer != nil {
		if err := patch(fieldManufacturer, 2, func(b []byte) { order.PutUint16(b, *s.Manufacturer) }); err != nil {
			return nil, err
		}
	}
	if s.Product != nil {
		if err := patch(fieldProduct, 2, func(b []byte) { order.PutUint16(b, *s.Product) }); err != nil {
			return nil, err
		}
	}
	if s.Serial != nil {
		if err := patch(fieldSerialNumber, 4, func(b []byte) { order.PutUint32(b, *s.Serial) }); err != nil {
			return nil, err
		}
	}

	// Recompute the trailing CRC over everything but the final 2 bytes.
	crc := crc16(out[:len(out)-2])
	binary.LittleEndian.PutUint16(out[len(out)-2:], crc)
	return out, nil
}
