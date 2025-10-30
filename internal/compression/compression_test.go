package compression

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompressData(t *testing.T) {
	testData := []byte("Hello, World! This is a test string for compression. " +
		"This string is long enough to be effectively compressed by gzip algorithm. " +
		"Repeating patterns help compression algorithms work better.")

	compressed, err := CompressData(testData)
	require.NoError(t, err, "Compression should succeed")

	assert.NotNil(t, compressed, "Compressed data should not be nil")
	assert.Less(t, len(compressed), len(testData), "Compressed data should be smaller than original")
	assert.Greater(t, len(compressed), 0, "Compressed data should not be empty")
}

func TestDecompressReader(t *testing.T) {
	originalData := []byte("Hello, World! This is a test string for compression.")

	compressed, err := CompressData(originalData)
	require.NoError(t, err, "Compression should succeed")

	decompressed, err := DecompressReader(bytes.NewReader(compressed))
	require.NoError(t, err, "Decompression should succeed")

	assert.Equal(t, originalData, decompressed, "Decompressed data should match original")
}

func TestDecompressReaderErrorHandling(t *testing.T) {
	invalidData := []byte("This is not gzip data")

	_, err := DecompressReader(bytes.NewReader(invalidData))
	assert.Error(t, err, "Should return error for invalid gzip data")
	assert.Contains(t, err.Error(), "failed to create gzip reader", "Error should mention gzip reader")
}

func TestCompressDecompressRoundTrip(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
	}{
		{"Empty data", []byte{}},
		{"Short string", []byte("Hello")},
		{"Long string", []byte("This is a very long string that should compress well with gzip compression algorithm")},
		{"JSON data", []byte(`{"id":"test","value":123.45,"delta":100}`)},
		{"Binary data", []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			compressed, err := CompressData(tc.data)
			require.NoError(t, err, "Compression should succeed")

			decompressed, err := DecompressData(compressed)
			require.NoError(t, err, "Decompression should succeed")

			assert.Equal(t, tc.data, decompressed, "Round trip should preserve original data")
		})
	}
}

func TestSupportsGzip(t *testing.T) {
	testCases := []struct {
		name           string
		acceptEncoding string
		expected       bool
	}{
		{"Empty header", "", false},
		{"Only gzip", "gzip", true},
		{"Gzip with deflate", "gzip, deflate", true},
		{"Deflate with gzip", "deflate, gzip", true},
		{"Only deflate", "deflate", false},
		{"Gzip with spaces", " gzip , deflate ", true},
		{"Multiple gzip", "gzip, gzip, deflate", true},
		{"Mixed case", "Gzip", false},
		{"With quality values", "gzip;q=0.8, deflate;q=0.6", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := SupportsGzip(tc.acceptEncoding)
			assert.Equal(t, tc.expected, result, "SupportsGzip should return correct result")
		})
	}
}

func TestGetCompressionStats(t *testing.T) {
	originalData := []byte("This is a test string for compression statistics. " +
		"This string contains repetitive patterns that should compress well. " +
		"Compression algorithms work best with redundant data patterns.")
	compressedData, err := CompressData(originalData)
	require.NoError(t, err, "Compression should succeed")

	stats := GetCompressionStats(originalData, compressedData)

	assert.Equal(t, len(originalData), stats.OriginalSize, "Original size should match")
	assert.Equal(t, len(compressedData), stats.CompressedSize, "Compressed size should match")
	assert.Less(t, stats.CompressionRatio, 1.0, "Compression ratio should be less than 1")
	assert.Greater(t, stats.CompressionRatio, 0.0, "Compression ratio should be greater than 0")
}

func TestCompressDataErrorHandling(t *testing.T) {
	largeData := make([]byte, 1024*1024*100)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	compressed, err := CompressData(largeData)
	if err != nil {
		t.Logf("Compression failed as expected for large data: %v", err)
	} else {
		decompressed, err := DecompressData(compressed)
		require.NoError(t, err, "Decompression should succeed")
		assert.Equal(t, largeData, decompressed, "Round trip should preserve large data")
	}
}
