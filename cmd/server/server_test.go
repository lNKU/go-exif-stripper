package main

import (
	"archive/zip"
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-exif-stripper/internal/exifstrip"
)

var fakeJPEG = []byte{
	0xFF, 0xD8, // SOI
	0xFF, 0xE0, 0x00, 0x04, 0x4A, 0x46, // APP0
	0xFF, 0xE1, 0x00, 0x04, 0x45, 0x78, // APP1 (fake EXIF)
	0xFF, 0xDA, // SOS
	0x12, 0x34, 0xFF, 0xD9, // fake compressed data + EOI
}

func TestIndexHandler(t *testing.T) {

	req := httptest.NewRequest("GET", "/", nil)

	rec := httptest.NewRecorder()

	indexHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `action="/upload"`) {
		t.Errorf("response body does not contain the expected upload form")
	}
}

func TestUploadHandler(t *testing.T) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	part, err := mw.CreateFormFile("photos", "test.jpg")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(fakeJPEG); err != nil {
		t.Fatalf("writing fake JPEG into form: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("closing multipart writer: %v", err)
	}

	req := httptest.NewRequest("POST", "/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	rec := httptest.NewRecorder()
	uploadHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/zip" {
		t.Errorf("Content-Type = %q, want application/zip", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Errorf("Content-Disposition = %q, want it to contain \"attachment\"", cd)
	}

	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("response body is not a valid zip: %v", err)
	}
	if len(zr.File) != 1 {
		t.Fatalf("zip contains %d file(s), want 1", len(zr.File))
	}

	entry := zr.File[0]
	if entry.Name != "test_stripped.jpg" {
		t.Errorf("entry name = %q, want %q", entry.Name, "test_stripped.jpg")
	}

	rc, err := entry.Open()
	if err != nil {
		t.Fatalf("opening zip entry: %v", err)
	}
	defer rc.Close()

	var out bytes.Buffer
	if _, err := out.ReadFrom(rc); err != nil {
		t.Fatalf("reading zip entry: %v", err)
	}

	segments, err := exifstrip.FindSegments(out.Bytes())
	if err != nil {
		t.Fatalf("FindSegments on stripped output: %v", err)
	}
	if exifstrip.HasEXIF(segments) {
		t.Errorf("stripped file in zip still contains EXIF data")
	}
}
