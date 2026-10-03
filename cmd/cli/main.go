package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
// exifstrip package, then writes the result alongside the original.
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

// outputPath derives an output filename by appending "_stripped" before
// the file extension, in the same directory as the original file.
func outputPath(inputPath string) string {
	dir := filepath.Dir(inputPath)   // e.g. "/home/user/Downloads"
	ext := filepath.Ext(inputPath)   // e.g. ".jpg", ".png", etc
	base := filepath.Base(inputPath) // e.g. "IMG_5836.JPG"
	nameOnly := strings.TrimSuffix(base, ext)
	return filepath.Join(dir, nameOnly+"_stripped"+ext)
}
