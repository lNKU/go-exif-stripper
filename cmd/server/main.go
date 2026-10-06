package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"

	"go-exif-stripper/internal/exifstrip"
)

const maxUploadSize = 32 << 20

func main() {
	http.HandleFunc("GET /{$}", indexHandler)
	http.HandleFunc("POST /upload", uploadHandler)

	addr := ":8080"
	fmt.Printf("Listening on http://localhost%s\n", addr)

	log.Fatal(http.ListenAndServe(addr, nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, indexHTML)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, fmt.Sprintf("could not parse upload - %v", err), http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		http.Error(w, "no files were uploaded", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="stripped_photos.zip"`)

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, fh := range files {
		if err := addStrippedFile(zw, fh); err != nil {
			log.Printf("skipping %s - %v", fh.Filename, err)
			continue
		}
	}
}

func addStrippedFile(zw *zip.Writer, fh *multipart.FileHeader) error {
	file, err := fh.Open()
	if err != nil {
		return fmt.Errorf("could not open upload - %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("could not read upload - %w", err)
	}

	if err := exifstrip.CheckJPEG(data); err != nil {
		return fmt.Errorf("not a valid JPEG - %w", err)
	}

	segments, err := exifstrip.FindSegments(data)
	if err != nil {
		return fmt.Errorf("error scanning segments - %w", err)
	}

	cleaned := data
	if exifstrip.HasEXIF(segments) {
		cleaned = exifstrip.StripEXIF(data, segments)
	}

	entry, err := zw.Create(exifstrip.StrippedName(fh.Filename))
	if err != nil {
		return fmt.Errorf("could not add to zip - %w", err)
	}

	_, err = entry.Write(cleaned)
	return err
}

const indexHTML = `<!DOCTYPE html>
<html>
<head>
	<title>EXIF Stripper</title>
</head>
<body>
	<h1>Strip EXIF data from files</h1>
	<form action="/upload" method="post" enctype="multipart/form-data">
		<input type="file" name="photos" multiple accept="image/jpeg">
		<button type="submit">Strip EXIF</button>
	</form>
</body>
</html>
`
