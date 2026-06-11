package fit

import (
	"encoding/binary"
	"testing"
)

func u16(v uint16) *uint16 { return &v }
func u32(v uint32) *uint32 { return &v }

// readFileID parses manufacturer/product/serial out of a FIT whose first record
// is a file_id definition + data message. It mirrors StampFile's parse and is
// used only to verify patched output.
func readFileID(t *testing.T, data []byte) (mfg, prod uint16, serial uint32) {
	t.Helper()
	headerSize := int(data[0])
	pos := headerSize
	pos++ // record header
	pos++ // reserved
	bigEndian := data[pos] != 0
	pos++ // architecture
	var order binary.ByteOrder = binary.LittleEndian
	if bigEndian {
		order = binary.BigEndian
	}
	pos += 2 // global msg num
	numFields := int(data[pos])
	pos++
	type loc struct{ off, size int }
	locs := map[int]loc{}
	dataOff := 0
	for i := 0; i < numFields; i++ {
		fn := int(data[pos])
		sz := int(data[pos+1])
		pos += 3
		locs[fn] = loc{dataOff, sz}
		dataOff += sz
	}
	dataStart := pos + 1 // skip data record header
	mfg = order.Uint16(data[dataStart+locs[fieldManufacturer].off:])
	prod = order.Uint16(data[dataStart+locs[fieldProduct].off:])
	serial = order.Uint32(data[dataStart+locs[fieldSerialNumber].off:])
	return mfg, prod, serial
}

// buildFIT constructs a minimal valid FIT: 14-byte header, one file_id
// definition + data message, and a correct trailing CRC.
func buildFIT(t *testing.T, mfg, prod uint16, serial uint32) []byte {
	t.Helper()
	// file_id definition: serial(3,uint32z), manufacturer(1,uint16),
	// product(2,uint16), type(0,enum).
	def := []byte{
		0x40,       // record header: definition, local 0
		0x00,       // reserved
		0x00,       // architecture: little-endian
		0x00, 0x00, // global msg num 0 (file_id)
		0x04,             // 4 fields
		0x03, 0x04, 0x8c, // serial_number, 4, uint32z
		0x01, 0x02, 0x84, // manufacturer, 2, uint16
		0x02, 0x02, 0x84, // product, 2, uint16
		0x00, 0x01, 0x00, // type, 1, enum
	}
	data := []byte{0x00} // data record header: data, local 0
	data = binary.LittleEndian.AppendUint32(data, serial)
	data = binary.LittleEndian.AppendUint16(data, mfg)
	data = binary.LittleEndian.AppendUint16(data, prod)
	data = append(data, 0x04) // type = activity

	body := append(def, data...)

	header := make([]byte, 14)
	header[0] = 14
	header[1] = 0x10
	binary.LittleEndian.PutUint16(header[2:], 2126)
	binary.LittleEndian.PutUint32(header[4:], uint32(len(body)))
	copy(header[8:], ".FIT")
	binary.LittleEndian.PutUint16(header[12:], crc16(header[:12]))

	file := append(header, body...)
	file = binary.LittleEndian.AppendUint16(file, crc16(file))
	return file
}

func assertCRCValid(t *testing.T, data []byte) {
	t.Helper()
	want := crc16(data[:len(data)-2])
	got := binary.LittleEndian.Uint16(data[len(data)-2:])
	if want != got {
		t.Fatalf("trailing CRC invalid: got 0x%04x want 0x%04x", got, want)
	}
}

// TestCRC16KnownAnswer guards the FIT CRC-16 against an independently computed
// oracle (the same algorithm verified to reproduce a real Garmin file's header
// and trailer CRCs during development).
func TestCRC16KnownAnswer(t *testing.T) {
	seq := make([]byte, 16)
	for i := range seq {
		seq[i] = byte(i)
	}
	if got := crc16(seq); got != 0x170a {
		t.Errorf("crc16(0x00..0x0F) = 0x%04x, want 0x170a", got)
	}
	if got := crc16([]byte("123456789")); got != 0xbb3d {
		t.Errorf("crc16(\"123456789\") = 0x%04x, want 0xbb3d", got)
	}
}

func TestStampSynthetic(t *testing.T) {
	orig := buildFIT(t, 1, 3290, 3309787599)
	assertCRCValid(t, orig)

	out, err := StampFile(orig, Stamp{Manufacturer: u16(1), Product: u16(3570), Serial: u32(3426524463)})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(orig) {
		t.Fatalf("size changed: %d -> %d", len(orig), len(out))
	}
	assertCRCValid(t, out)
	mfg, prod, serial := readFileID(t, out)
	if mfg != 1 || prod != 3570 || serial != 3426524463 {
		t.Fatalf("file_id not patched: mfg=%d prod=%d serial=%d", mfg, prod, serial)
	}
	// Input must be untouched.
	if im, ip, is := readFileID(t, orig); im != 1 || ip != 3290 || is != 3309787599 {
		t.Fatalf("input mutated: mfg=%d prod=%d serial=%d", im, ip, is)
	}
}

func TestStampPartial(t *testing.T) {
	orig := buildFIT(t, 1, 3290, 100)
	out, err := StampFile(orig, Stamp{Serial: u32(999)})
	if err != nil {
		t.Fatal(err)
	}
	assertCRCValid(t, out)
	mfg, prod, serial := readFileID(t, out)
	if mfg != 1 || prod != 3290 || serial != 999 {
		t.Fatalf("partial stamp wrong: mfg=%d prod=%d serial=%d", mfg, prod, serial)
	}
}

func TestStampNoChange(t *testing.T) {
	orig := buildFIT(t, 1, 3290, 100)
	out, err := StampFile(orig, Stamp{})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(orig) {
		t.Fatal("zero stamp should return identical bytes")
	}
}

func TestStampErrors(t *testing.T) {
	good := buildFIT(t, 1, 3290, 100)
	cases := map[string][]byte{
		"too short":     []byte{0x0e, 0x10},
		"bad signature": append([]byte{14, 0x10, 0, 0, 0, 0, 0, 0, 'N', 'O', 'P', 'E'}, make([]byte, 10)...),
	}
	for name, data := range cases {
		if _, err := StampFile(data, Stamp{Serial: u32(1)}); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	// First record not a definition: flip the def header to a data header.
	bad := make([]byte, len(good))
	copy(bad, good)
	bad[14] = 0x00
	if _, err := StampFile(bad, Stamp{Serial: u32(1)}); err == nil {
		t.Error("non-definition first record: expected error")
	}
}

func TestStampMissingField(t *testing.T) {
	// file_id with only manufacturer + type, no serial field.
	def := []byte{0x40, 0x00, 0x00, 0x00, 0x00, 0x02, 0x01, 0x02, 0x84, 0x00, 0x01, 0x00}
	data := []byte{0x00, 0x01, 0x00, 0x04}
	body := append(def, data...)
	header := make([]byte, 14)
	header[0] = 14
	copy(header[8:], ".FIT")
	binary.LittleEndian.PutUint32(header[4:], uint32(len(body)))
	binary.LittleEndian.PutUint16(header[12:], crc16(header[:12]))
	file := append(header, body...)
	file = binary.LittleEndian.AppendUint16(file, crc16(file))

	if _, err := StampFile(file, Stamp{Serial: u32(1)}); err == nil {
		t.Fatal("expected error stamping absent serial field")
	}
}
