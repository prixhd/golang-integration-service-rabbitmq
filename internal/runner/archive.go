package runner

import (
	"archive/tar"
	"bytes"
	"fmt"
)

func createCodeArchive(filename string, code string) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	tw := tar.NewWriter(&buf)

	header := &tar.Header{
		Name: filename,
		Mode: 0644,
		Size: int64(len(code)),
	}

	if err := tw.WriteHeader(header); err != nil {
		return nil, fmt.Errorf("failed to write tar header: %w", err)
	}

	if _, err := tw.Write([]byte(code)); err != nil {
		return nil, fmt.Errorf("failed to write code to tar: %w", err)
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close tar writer: %w", err)
	}

	return &buf, nil
}
