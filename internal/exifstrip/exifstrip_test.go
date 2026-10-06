package exifstrip

import (
	"bytes"
	"testing"
)

func TestCheckJPEG(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid SOI marker",
			data:    []byte{0xFF, 0xD8, 0xFF, 0xE0},
			wantErr: false,
		},
		{
			name:    "too short to contain a marker",
			data:    []byte{0xFF},
			wantErr: true,
		},
		{
			name:    "wrong bytes at the start",
			data:    []byte{0x00, 0x00, 0xFF, 0xD8},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckJPEG(tt.data)
			gotErr := err != nil
			if gotErr != tt.wantErr {
				t.Errorf("\nCheckJPEG(%v) error = %v, wantErr %v", tt.data, err, tt.wantErr)
			}
		})
	}
}

func TestFindSegments(t *testing.T) {
	data := []byte{
		0xFF, 0xD8, // SOI
		0xFF, 0xE0, 0x00, 0x04, 0x4A, 0x46, // APP0: length=4 (2 length bytes + 2 payload bytes)
		0xFF, 0xE1, 0x00, 0x04, 0x45, 0x78, // APP1: length=4 (fake EXIF payload)
		0xFF, 0xDA, // SOS
		0x12, 0x34, 0xFF, 0xD9, // fake compressed data + EOI
	}

	got, err := FindSegments(data)
	if err != nil {
		t.Fatalf("\nFindSegments returned unexpected error: %v", err)
	}

	want := []Segment{
		{Marker: 0xE0, Offset: 2, Length: 4},
		{Marker: 0xE1, Offset: 8, Length: 4},
		{Marker: 0xDA, Offset: 14, Length: 0},
	}

	if len(got) != len(want) {
		t.Fatalf("\nGot %d segments, want %d\ngot:  %+v\nwant: %+v", len(got), len(want), got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("\nSegment %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestStrippedName(t *testing.T) {
	got := StrippedName("IMG_5836.JPG")
	want := "IMG_5836_stripped.JPG"
	if got != want {
		t.Errorf("StrippedName(%q) = %q, want %q", "IMG_5836.JPG", got, want)
	}
}

func TestHasEXIF(t *testing.T) {
	withEXIF := []Segment{{Marker: 0xE0}, {Marker: 0xE1}, {Marker: 0xDA}}
	withoutEXIF := []Segment{{Marker: 0xE0}, {Marker: 0xDA}}

	if !HasEXIF(withEXIF) {
		t.Errorf("HasEXIF(%+v) = false, want true", withEXIF)
	}
	if HasEXIF(withoutEXIF) {
		t.Errorf("HasEXIF(%+v) = true, want false", withoutEXIF)
	}
}

func TestStripEXIF(t *testing.T) {
	data := []byte{
		0xFF, 0xD8,
		0xFF, 0xE0, 0x00, 0x04, 0x4A, 0x46,
		0xFF, 0xE1, 0x00, 0x04, 0x45, 0x78,
		0xFF, 0xDA,
		0x12, 0x34, 0xFF, 0xD9,
	}

	segments, err := FindSegments(data)
	if err != nil {
		t.Fatalf("\nFindSegments returned unexpected error: %v", err)
	}

	got := StripEXIF(data, segments)

	want := []byte{
		0xFF, 0xD8,
		0xFF, 0xE0, 0x00, 0x04, 0x4A, 0x46,
		0xFF, 0xDA,
		0x12, 0x34, 0xFF, 0xD9,
	}

	if !bytes.Equal(got, want) {
		t.Errorf("\nStripEXIF result mismatch\ngot:  %#x\nwant: %#x", got, want)
	}
}
