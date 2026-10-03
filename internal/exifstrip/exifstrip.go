package exifstrip

import (
	"encoding/binary"
	"fmt"
)

// verifies that data begins with valid "Start of Image" bytes
func CheckJPEG(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("File is too small to be a JPEG.")
	}
	if data[0] != 0xFF || data[1] != 0xD8 {
		return fmt.Errorf("Missing JPEG Start of Image marker (got %#x %#x).", data[0], data[1])
	}
	return nil
}

// segment describes one marker segment found in a JPEG file
type Segment struct {
	Marker byte
	Offset int
	Length int
}

// walks through a file's marker segments, starting right after the
// SOI marker, and returns each one sequentially.
func FindSegments(data []byte) ([]Segment, error) {
	var segments []Segment

	i := 2
	for i < len(data)-1 {
		if data[i] != 0xFF {
			return nil, fmt.Errorf("Expected marker byte 0xFF at offset %d, got %#x.", i, data[i])
		}
		marker := data[i+1]

		if marker == 0xDA {
			segments = append(segments, Segment{Marker: marker, Offset: i})
			break
		}

		if i+4 > len(data) {
			return nil, fmt.Errorf("Truncated segment at offset %d.", i)
		}

		length := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		segments = append(segments, Segment{Marker: marker, Offset: i, Length: length})
		i += 2 + length
	}

	return segments, nil
}

// rebuilds a file's bytes with every APP1 marker segment removed.
func StripEXIF(data []byte, segments []Segment) []byte {
	result := make([]byte, 0, len(data))
	result = append(result, data[0:2]...)

	for _, seg := range segments {
		if seg.Marker == 0xE1 {
			continue
		}

		if seg.Marker == 0xDA {
			result = append(result, data[seg.Offset:]...)
			break
		}

		end := seg.Offset + 2 + seg.Length
		result = append(result, data[seg.Offset:end]...)
	}

	return result
}
