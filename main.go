package main
 
import (
	"encoding/binary"
	"fmt"
	"os"
)
 
func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: inspect <path-to-file>")
		os.Exit(1)
	}
 
	path := os.Args[1]
 
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("could not read file: %v\n", err)
		os.Exit(1)
	}
 
	fmt.Printf("File: %s\n", path)
	fmt.Printf("Size: %d bytes\n", len(data))

	if err := checkJPEG(data); err != nil {
		fmt.Printf("Invalid JPEG: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Valid JPEG image.")

	segments, err := findSegments(data)
	if err != nil {
		fmt.Printf("error scanning segments: %v\n", err)
		os.Exit(1)
	}

	for _, seg := range segments {
		tag := ""
		if seg.marker == 0xE1 {
			tag = "  <-- APP1 (likely EXIF or XMP metadata)"
		}
		fmt.Printf("marker 0xFF%02X at offset %d, length %d%s\n", seg.marker, seg.offset, seg.length, tag)
	}
}

// checkJPEG verifies that data begins with the JPEG "Start of Image" (SOI)
// marker: the two bytes 0xFF, 0xD8. If those aren't there, it's not a JPEG
// (or the file is corrupted/truncated).
//
// This function only returns an error - it never calls os.Exit or prints
// anything. Keeping "detect a problem" separate from "decide what to do
// about it" means main() can react one way (print and quit), while a
// future web server reacts another way (send an HTTP error), reusing this
// exact function unchanged.
func checkJPEG(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("file is too small to be a JPEG")
	}
	if data[0] != 0xFF || data[1] != 0xD8 {
		return fmt.Errorf("missing JPEG SOI marker (got %#x %#x)", data[0], data[1])
	}
	return nil
}

// segment describes one marker segment found in a JPEG file: which marker
// it is, where it starts (the offset of its 0xFF byte), and how long it is
// according to its own length field.
//
// Bundling these three related values into one named type, instead of
// tracking three separate slices (markers, offsets, lengths), is what a
// struct is for: values that only make sense together travel together.
type segment struct {
	marker byte
	offset int
	length int
}

// findSegments walks a JPEG's marker segments, starting right after the
// SOI marker, and returns each one it finds up to (and including) the
// Start of Scan (SOS) marker, where segment headers end and compressed
// image data begins.
func findSegments(data []byte) ([]segment, error) {
	var segments []segment // starts as nil; append() below grows it as needed
 
	i := 2 // skip the 2-byte SOI marker checkJPEG already validated
	for i < len(data)-1 {
		if data[i] != 0xFF {
			return nil, fmt.Errorf("expected marker byte 0xFF at offset %d, got %#x", i, data[i])
		}
		marker := data[i+1]
 
		// SOS begins the compressed image data - no more segments follow it,
		// so record it and stop walking.
		if marker == 0xDA {
			segments = append(segments, segment{marker: marker, offset: i})
			break
		}
 
		if i+4 > len(data) {
			return nil, fmt.Errorf("truncated segment at offset %d", i)
		}
 
		// The 2 bytes after the marker are a big-endian uint16: the segment's
		// length, INCLUDING these 2 length bytes (but not the marker itself).
		// encoding/binary.BigEndian.Uint16 reads those 2 bytes as a number
		// the same way the JPEG spec defines - most significant byte first.
		length := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
 
		segments = append(segments, segment{marker: marker, offset: i, length: length})
 
		// Advance past: the marker itself (2 bytes) + everything the length
		// field says belongs to this segment.
		i += 2 + length
	}
 
	return segments, nil
}
