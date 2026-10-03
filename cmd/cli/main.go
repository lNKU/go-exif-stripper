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

	var successes, failed int
	for _, path := range paths {
		if err := processFile(path); err != nil {
			fmt.Printf("%s: FAILED: %v\n", path, err)
			failed++
			continue
		}
		successes++
	}
	fmt.Printf("FINISHED: %d succeeded, %d failed (of %d total)\n", successes, failed, len(paths))
}

// processFile reads file from disk, strips the EXIF data using the
// exifstrip package, then writes the result alongside the original.
func processFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("File could not be read: %w", err)
	}

	if err := exifstrip.CheckJPEG(data); err != nil {
		return fmt.Errorf("Invalid JPEG: %w", err)
	}

	segments, err := exifstrip.FindSegments(data)
	if err != nil {
		return fmt.Errorf("Error scanning segments: %w", err)
	}

	cleaned := exifstrip.StripEXIF(data, segments)
	outPath := outputPath(path)
	if err := os.WriteFile(outPath, cleaned, 0644); err != nil {
		return fmt.Errorf("could not write output file: %w", err)
	}

	fmt.Printf("%s -> %s (%d bytes, removed %d bytes)\n",
		path, outPath, len(cleaned), len(data)-len(cleaned))
	return nil
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
