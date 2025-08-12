// mmso-go: Motorsport Manager save edit suite
// Copyright (C) 2018  Yishen Miao
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package mmse_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/pierrec/lz4"
	"github.com/stretchr/testify/assert"

	"github.com/mys721tx/mmse-go/pkg/mmse"
)

// Test data and helpers
var (
	testInt32Value         = int32(0x7fffffff)
	testInt32Bytes         = []byte{0xff, 0xff, 0xff, 0x7f}
	testNegativeInt32      = int32(-1)
	testNegativeInt32Bytes = []byte{0xff, 0xff, 0xff, 0xff}
	testZeroInt32          = int32(0)
	testZeroInt32Bytes     = []byte{0x00, 0x00, 0x00, 0x00}
)

// Mock types for testing error conditions
type errorReader struct {
	err error
}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, r.err
}

type errorWriter struct {
	err error
}

func (w *errorWriter) Write(p []byte) (int, error) {
	return 0, w.err
}

type partialReader struct {
	data []byte
	pos  int
}

func (r *partialReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}

	// Only read one byte at a time to simulate partial reads
	n := copy(p, r.data[r.pos:r.pos+1])
	r.pos += n
	return n, nil
}

// TestReadInt32 tests the ReadInt32 function with various inputs
func TestReadInt32(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected int32
		wantErr  bool
	}{
		{
			name:     "positive max int32",
			input:    testInt32Bytes,
			expected: testInt32Value,
			wantErr:  false,
		},
		{
			name:     "negative int32",
			input:    testNegativeInt32Bytes,
			expected: testNegativeInt32,
			wantErr:  false,
		},
		{
			name:     "zero int32",
			input:    testZeroInt32Bytes,
			expected: testZeroInt32,
			wantErr:  false,
		},
		{
			name:     "insufficient data",
			input:    []byte{0xff, 0xff},
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "empty input",
			input:    []byte{},
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.input)
			got, err := mmse.ReadInt32(r)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, int32(0), got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestReadInt32WithPartialRead(t *testing.T) {
	// Test with a reader that returns data in small chunks
	r := &partialReader{data: testInt32Bytes}
	got, err := mmse.ReadInt32(r)

	assert.NoError(t, err)
	assert.Equal(t, testInt32Value, got)
}

func TestReadInt32WithReaderError(t *testing.T) {
	testErrors := []error{
		io.EOF,
		io.ErrUnexpectedEOF,
		errors.New("custom read error"),
	}

	for _, testErr := range testErrors {
		t.Run(testErr.Error(), func(t *testing.T) {
			r := &errorReader{err: testErr}
			got, err := mmse.ReadInt32(r)

			assert.Error(t, err)
			assert.Equal(t, testErr, err)
			assert.Equal(t, int32(0), got)
		})
	}
}

// TestWriteInt32 tests the WriteInt32 function
func TestWriteInt32(t *testing.T) {
	tests := []struct {
		name     string
		value    int32
		expected []byte
	}{
		{
			name:     "positive max int32",
			value:    testInt32Value,
			expected: testInt32Bytes,
		},
		{
			name:     "negative int32",
			value:    testNegativeInt32,
			expected: testNegativeInt32Bytes,
		},
		{
			name:     "zero int32",
			value:    testZeroInt32,
			expected: testZeroInt32Bytes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := mmse.WriteInt32(&buf, tt.value)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, buf.Bytes())
		})
	}
}

func TestWriteInt32WithWriterError(t *testing.T) {
	testErrors := []error{
		os.ErrPermission,
		os.ErrNotExist,
		os.ErrClosed,
		io.ErrShortWrite,
		errors.New("custom write error"),
	}

	for _, testErr := range testErrors {
		t.Run(testErr.Error(), func(t *testing.T) {
			w := &errorWriter{err: testErr}
			err := mmse.WriteInt32(w, testInt32Value)

			assert.Error(t, err)
			assert.Equal(t, testErr, err)
		})
	}
}

// TestReadSizeToFrame tests the ReadSizeToFrame function
func TestReadSizeToFrame(t *testing.T) {
	tests := []struct {
		name    string
		sizeCom int32
		sizeRaw int32
	}{
		{
			name:    "normal sizes",
			sizeCom: 1000,
			sizeRaw: 2000,
		},
		{
			name:    "equal sizes",
			sizeCom: 500,
			sizeRaw: 500,
		},
		{
			name:    "zero sizes",
			sizeCom: 0,
			sizeRaw: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			binary.Write(&buf, binary.LittleEndian, tt.sizeCom)
			binary.Write(&buf, binary.LittleEndian, tt.sizeRaw)

			frame := mmse.ReadSizeToFrame(&buf)

			assert.Equal(t, tt.sizeCom, frame.SizeCom)
			assert.Equal(t, tt.sizeRaw, frame.SizeRaw)
		})
	}
}

func TestReadSizeToFrameWithReaderError(t *testing.T) {
	testErrors := []error{
		io.EOF,
		io.ErrUnexpectedEOF,
		errors.New("custom read error"),
	}

	for _, testErr := range testErrors {
		t.Run("error reading SizeCom: "+testErr.Error(), func(t *testing.T) {
			r := &errorReader{err: testErr}

			assert.Panics(t, func() {
				mmse.ReadSizeToFrame(r)
			})
		})

		t.Run("error reading SizeRaw: "+testErr.Error(), func(t *testing.T) {
			var buf bytes.Buffer
			binary.Write(&buf, binary.LittleEndian, int32(100)) // Write valid SizeCom

			// Create a reader that fails on second read
			combinedReader := io.MultiReader(&buf, &errorReader{err: testErr})

			assert.Panics(t, func() {
				mmse.ReadSizeToFrame(combinedReader)
			})
		})
	}
}

// TestFrame tests the Frame type methods
func TestFrameEncode(t *testing.T) {
	tests := []struct {
		name       string
		data       string
		wantErr    bool
		errMessage string
	}{
		{
			name:    "encode simple data",
			data:    "Hello, World!",
			wantErr: false,
		},
		{
			name:    "encode empty data",
			data:    "",
			wantErr: false,
		},
		{
			name:    "encode large data",
			data:    string(bytes.Repeat([]byte("A"), 10000)),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := &mmse.Frame{}
			frame.WriteString(tt.data)
			frame.SizeRaw = int32(len(tt.data))

			err := frame.Encode()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.True(t, frame.SizeCom > 0 || len(tt.data) == 0)
				assert.Equal(t, int32(len(tt.data)), frame.SizeRaw)
			}
		})
	}
}

func TestFrameEncodeAlreadyEncoded(t *testing.T) {
	frame := &mmse.Frame{}
	frame.WriteString("test data")
	frame.SizeRaw = 9

	// First encode should succeed
	err := frame.Encode()
	assert.NoError(t, err)

	// Second encode should fail
	err = frame.Encode()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already encoded")
}

func TestFrameDecode(t *testing.T) {
	// Test data that we know will compress well
	originalData := string(bytes.Repeat([]byte("A"), 100)) // Repeating pattern compresses well

	// Create frame using ReadSizeToFrame (which sets isEncoded = true)
	var sizeBuf bytes.Buffer
	binary.Write(&sizeBuf, binary.LittleEndian, int32(0))                 // SizeCom (will be updated)
	binary.Write(&sizeBuf, binary.LittleEndian, int32(len(originalData))) // SizeRaw

	frame := mmse.ReadSizeToFrame(&sizeBuf)

	// Compress the data manually
	compressedData := make([]byte, len(originalData)*2) // Make buffer larger to ensure space
	n, err := lz4.CompressBlock([]byte(originalData), compressedData, make([]int, 1<<16))
	requireNoError(t, err)

	if n == 0 {
		// Data not compressible, use original
		frame.Write([]byte(originalData))
		frame.SizeCom = int32(len(originalData))
	} else {
		// Data compressed successfully
		frame.Write(compressedData[:n])
		frame.SizeCom = int32(n)
	}

	// Now decode it back
	err = frame.Decode()
	assert.NoError(t, err)

	decodedData := frame.String()
	assert.Equal(t, originalData, decodedData)
}

func TestFrameDecodeNotEncoded(t *testing.T) {
	frame := &mmse.Frame{}
	frame.WriteString("test data")

	err := frame.Decode()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not encoded")
}

func TestFrameDecodeCorruptedData(t *testing.T) {
	frame := &mmse.Frame{}
	// Write some invalid compressed data
	frame.Write([]byte{0x01, 0x02, 0x03, 0x04})
	frame.SizeRaw = 100 // Much larger than what the corrupt data would decompress to
	frame.SizeCom = 4
	// Manually set as encoded since we're simulating corrupted data
	// Note: We can't directly access isEncoded, so we'll create this through ReadSizeToFrame

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, int32(4))   // SizeCom
	binary.Write(&buf, binary.LittleEndian, int32(100)) // SizeRaw

	corruptedFrame := mmse.ReadSizeToFrame(&buf)
	corruptedFrame.Write([]byte{0x01, 0x02, 0x03, 0x04})

	err := corruptedFrame.Decode()
	assert.Error(t, err)
}

// TestCheckHeader tests the CheckHeader function
func TestCheckHeader(t *testing.T) {
	t.Run("valid header", func(t *testing.T) {
		var buf bytes.Buffer
		binary.Write(&buf, binary.LittleEndian, mmse.Magic)
		binary.Write(&buf, binary.LittleEndian, mmse.Ver)

		assert.NotPanics(t, func() {
			mmse.CheckHeader(&buf)
		})
	})

	t.Run("invalid magic number", func(t *testing.T) {
		var buf bytes.Buffer
		binary.Write(&buf, binary.LittleEndian, int32(0x12345678)) // Wrong magic
		binary.Write(&buf, binary.LittleEndian, mmse.Ver)

		assert.Panics(t, func() {
			mmse.CheckHeader(&buf)
		})
	})

	t.Run("invalid version number", func(t *testing.T) {
		var buf bytes.Buffer
		binary.Write(&buf, binary.LittleEndian, mmse.Magic)
		binary.Write(&buf, binary.LittleEndian, int32(0x12345678)) // Wrong version

		assert.Panics(t, func() {
			mmse.CheckHeader(&buf)
		})
	})

	t.Run("read error on magic", func(t *testing.T) {
		r := &errorReader{err: io.EOF}

		assert.Panics(t, func() {
			mmse.CheckHeader(r)
		})
	})

	t.Run("read error on version", func(t *testing.T) {
		var buf bytes.Buffer
		binary.Write(&buf, binary.LittleEndian, mmse.Magic)

		combinedReader := io.MultiReader(&buf, &errorReader{err: io.EOF})

		assert.Panics(t, func() {
			mmse.CheckHeader(combinedReader)
		})
	})
}

// TestWriteHeader tests the WriteHeader function
func TestWriteHeader(t *testing.T) {
	t.Run("successful write", func(t *testing.T) {
		var buf bytes.Buffer

		assert.NotPanics(t, func() {
			mmse.WriteHeader(&buf)
		})

		// Verify the written data
		var magic, version int32
		binary.Read(&buf, binary.LittleEndian, &magic)
		binary.Read(&buf, binary.LittleEndian, &version)

		assert.Equal(t, mmse.Magic, magic)
		assert.Equal(t, mmse.Ver, version)
	})

	t.Run("write error on magic", func(t *testing.T) {
		w := &errorWriter{err: errors.New("write error")}

		assert.Panics(t, func() {
			mmse.WriteHeader(w)
		})
	})
}

// TestWriteSize tests the WriteSize function
func TestWriteSize(t *testing.T) {
	t.Run("successful write", func(t *testing.T) {
		var buf bytes.Buffer
		frame := &mmse.Frame{}
		frame.SizeCom = 100
		frame.SizeRaw = 200

		assert.NotPanics(t, func() {
			mmse.WriteSize(&buf, frame)
		})

		// Verify the written data
		var sizeCom, sizeRaw int32
		binary.Read(&buf, binary.LittleEndian, &sizeCom)
		binary.Read(&buf, binary.LittleEndian, &sizeRaw)

		assert.Equal(t, frame.SizeCom, sizeCom)
		assert.Equal(t, frame.SizeRaw, sizeRaw)
	})

	t.Run("write error on sizeCom", func(t *testing.T) {
		w := &errorWriter{err: errors.New("write error")}
		frame := &mmse.Frame{}

		assert.Panics(t, func() {
			mmse.WriteSize(w, frame)
		})
	})
}

// TestWriteFrame tests the WriteFrame function
func TestWriteFrame(t *testing.T) {
	t.Run("successful write", func(t *testing.T) {
		var buf bytes.Buffer
		frame := &mmse.Frame{}
		testData := "Hello, World!"
		frame.WriteString(testData)

		assert.NotPanics(t, func() {
			mmse.WriteFrame(&buf, frame)
		})

		assert.Equal(t, testData, buf.String())
	})

	t.Run("write error", func(t *testing.T) {
		w := &errorWriter{err: errors.New("write error")}
		frame := &mmse.Frame{}
		frame.WriteString("test data")

		assert.Panics(t, func() {
			mmse.WriteFrame(w, frame)
		})
	})
}

// TestWriteJSON tests the WriteJSON function with temporary files
func TestWriteJSON(t *testing.T) {
	t.Run("successful write", func(t *testing.T) {
		// Create a frame with encoded data using ReadSizeToFrame approach
		// Use data that compresses well
		originalData := string(bytes.Repeat([]byte("A"), 50)) + `{"test": "data", "number": 42}`

		// Compress the data manually
		compressedData := make([]byte, len(originalData)*2)
		n, err := lz4.CompressBlock([]byte(originalData), compressedData, make([]int, 1<<16))
		requireNoError(t, err)

		sizeCom := int32(len(originalData)) // Default to uncompressed size
		finalData := []byte(originalData)   // Default to original data

		if n > 0 {
			// Compression succeeded
			sizeCom = int32(n)
			finalData = compressedData[:n]
		}

		// Create frame using ReadSizeToFrame
		var sizeBuf bytes.Buffer
		binary.Write(&sizeBuf, binary.LittleEndian, sizeCom)
		binary.Write(&sizeBuf, binary.LittleEndian, int32(len(originalData)))

		frame := mmse.ReadSizeToFrame(&sizeBuf)

		// Create a reader with the compressed frame data
		dataReader := bytes.NewReader(finalData)

		// Create temp file
		tmpFile, err := os.CreateTemp("", "test_*.json")
		requireNoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		assert.NotPanics(t, func() {
			mmse.WriteJSON(tmpFile.Name(), dataReader, frame)
		})

		// Verify the file content
		content, err := os.ReadFile(tmpFile.Name())
		requireNoError(t, err)
		assert.Equal(t, originalData, string(content))
	})

	t.Run("read error", func(t *testing.T) {
		frame := &mmse.Frame{}
		frame.SizeCom = 100 // Expect to read 100 bytes
		r := &errorReader{err: io.EOF}

		tmpFile, err := os.CreateTemp("", "test_*.json")
		requireNoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		assert.Panics(t, func() {
			mmse.WriteJSON(tmpFile.Name(), r, frame)
		})
	})
} // TestReadJSONToFrame tests the ReadJSONToFrame function with temporary files
func TestReadJSONToFrame(t *testing.T) {
	t.Run("successful read", func(t *testing.T) {
		testData := `{"test": "data", "array": [1, 2, 3]}`

		// Create temp file with test data
		tmpFile, err := os.CreateTemp("", "test_*.json")
		requireNoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(testData)
		requireNoError(t, err)
		tmpFile.Close()

		var frame *mmse.Frame
		assert.NotPanics(t, func() {
			frame = mmse.ReadJSONToFrame(tmpFile.Name())
		})

		assert.Equal(t, int32(len(testData)), frame.SizeRaw)
		assert.True(t, frame.SizeCom > 0)
	})

	t.Run("file not found", func(t *testing.T) {
		assert.Panics(t, func() {
			mmse.ReadJSONToFrame("/nonexistent/file.json")
		})
	})
}

// Helper function that we need to add since it's not available
func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// TestMagicAndVersionConstants tests that the constants are correctly defined
func TestMagicAndVersionConstants(t *testing.T) {
	assert.Equal(t, int32(0x73326d6d), mmse.Magic, "Magic constant should be 0x73326d6d")
	assert.Equal(t, int32(0x00000004), mmse.Ver, "Version constant should be 0x00000004")
}
