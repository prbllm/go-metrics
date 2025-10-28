package compression

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
)

func CompressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)

	_, err := gzWriter.Write(data)
	if err != nil {
		if closeErr := gzWriter.Close(); closeErr != nil {
			return nil, fmt.Errorf("failed to close gzip writer: %w", closeErr)
		}
		return nil, fmt.Errorf("failed to write to gzip writer: %w", err)
	}

	err = gzWriter.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

func DecompressData(data []byte) ([]byte, error) {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	decompressedData, err := io.ReadAll(gzReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress gzip data: %w", err)
	}

	return decompressedData, nil
}

func DecompressReader(reader io.Reader) ([]byte, error) {
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	decompressedData, err := io.ReadAll(gzReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress gzip data: %w", err)
	}

	return decompressedData, nil
}

func SupportsGzip(acceptEncoding string) bool {
	if acceptEncoding == "" {
		return false
	}

	encodings := strings.Split(acceptEncoding, ",")
	for _, encoding := range encodings {
		encoding = strings.TrimSpace(encoding)
		if idx := strings.Index(encoding, ";"); idx != -1 {
			encoding = encoding[:idx]
		}
		if encoding == "gzip" {
			return true
		}
	}
	return false
}

type CompressionStats struct {
	OriginalSize     int
	CompressedSize   int
	CompressionRatio float64
}

func GetCompressionStats(original, compressed []byte) CompressionStats {
	originalSize := len(original)
	compressedSize := len(compressed)

	var ratio float64
	if originalSize > 0 {
		ratio = float64(compressedSize) / float64(originalSize)
	}

	return CompressionStats{
		OriginalSize:     originalSize,
		CompressedSize:   compressedSize,
		CompressionRatio: ratio,
	}
}
