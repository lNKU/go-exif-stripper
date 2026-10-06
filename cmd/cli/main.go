package main

import (
	"fmt"
	"os"
	"path/filepath"

	"go-exif-stripper/internal/exifstrip"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: cli <path-to-file> [more-files...]")
		os.Exit(1)
	}

	paths := os.Args[1:]

	var stripped, skipped, failed int
	for _, path := range paths {
		didStrip, err := processFile(path)
		if err != nil {
			fmt.Printf("%s: FAILURE: %v\n", path, err)
			failed++
			continue
		}
		if didStrip {
			stripped++
		} else {
			skipped++
		}
	}

	fmt.Printf("\nSUCCESS: %d EXIF removed, %d already clean, %d failed (of %d total)\n",
		stripped, skipped, failed, len(paths))
}

// processFile reads file from disk, strips the EXIF data using the
// exifstrip, then writes the result beside the original.
func processFile(path string) (stripped bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("could not read file: %w", err)
	}

	if err := exifstrip.CheckJPEG(data); err != nil {
		return false, fmt.Errorf("not a valid JPEG: %w", err)
	}

	segments, err := exifstrip.FindSegments(data)
	if err != nil {
		return false, fmt.Errorf("error scanning segments: %w", err)
	}

	if !exifstrip.HasEXIF(segments) {
		fmt.Printf("%s: No EXIF data found, nothing to remove.\n", path)
		return false, nil
	}

	cleaned := exifstrip.StripEXIF(data, segments)

	outPath := outputPath(path)
	if err := os.WriteFile(outPath, cleaned, 0644); err != nil {
		return false, fmt.Errorf("could not write output file: %w", err)
	}

	fmt.Printf("%s -> %s (%d bytes, removed %d bytes)\n",
		path, outPath, len(cleaned), len(data)-len(cleaned))
	return true, nil
}

// outputPath derives an output filename by utilizing StrippedName's ExifStrip func
// to apply naming nomenclature to file(s) within archive
func outputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	name := exifstrip.StrippedName(filepath.Base(inputPath))
	return filepath.Join(dir, name)
}
