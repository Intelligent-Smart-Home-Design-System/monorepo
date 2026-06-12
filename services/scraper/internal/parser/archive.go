package parser

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

type ArchiveFile struct {
	Name string
	Data []byte
}

// decompresses a .tar.gz blob and returns all contained files
func extractArchive(raw []byte) ([]*ArchiveFile, error) {
	gr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	var files []*ArchiveFile

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar next: %w", err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("read tar entry %s: %w", hdr.Name, err)
		}
		files = append(files, &ArchiveFile{Name: hdr.Name, Data: data})
	}
	return files, nil
}

// returns the data of the first entry matching name, or an error.
func FindFile(files []*ArchiveFile, name string) ([]byte, error) {
	for _, f := range files {
		if f.Name == name {
			return f.Data, nil
		}
	}
	return nil, fmt.Errorf("file %q not found in archive", name)
}
