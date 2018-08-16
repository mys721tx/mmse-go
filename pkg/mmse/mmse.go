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

package mmse

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/pierrec/lz4"
)

const (
	// Magic is the magic number for Motorsport Manager save files.
	Magic int32 = 0x73326d6d
	// Ver is the version number for Motorsport Manager save files.
	Ver int32 = 0x00000004
)

// Frame provides storage for lz4 by embedding bytes.Buffer.
type Frame struct {
	SizeRaw   int32
	SizeCom   int32
	isEncoded bool
	bytes.Buffer
}

// Decode decodes the frame content in place. Decode will return error when
// isEncoded is false.
func (f *Frame) Decode() error {
	if !f.isEncoded {
		return fmt.Errorf("Frame is not encoded")
	}

	b := make([]byte, f.SizeRaw)

	n, err := lz4.UncompressBlock(f.Bytes(), b)

	if err != nil {
		return err
	}

	if int32(n) != f.SizeRaw {
		return fmt.Errorf(
			"expecting %d bytes, read %d",
			f.SizeRaw, int32(n),
		)
	}

	f.Reset()

	_, err = f.Write(b)

	f.isEncoded = false

	return err
}

// Encode encodes the frame content in place. Encode will return error when
// isEncoded is true.
func (f *Frame) Encode() error {
	if f.isEncoded {
		return fmt.Errorf("Frame is already encoded")
	}

	b := make([]byte, f.SizeRaw)

	n, err := lz4.CompressBlock(f.Bytes(), b, make([]int, 1<<16))

	if err != nil {
		return err
	}

	// lz4.CompressBLock returns 0 if the data is not compressible.
	if n == 0 {
		f.SizeCom = f.SizeRaw
	} else {
		f.SizeCom = int32(n)
	}

	f.Reset()

	_, err = f.Write(b)

	f.isEncoded = true

	f.Truncate(int(f.SizeCom))

	return err
}

// ReadInt32 reads an int32 from a file.
func ReadInt32(r io.Reader) (int32, error) {
	var v int32

	if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
		return 0, err
	}

	return v, nil
}

// WriteInt32 writes an int32 from a file.
func WriteInt32(w io.Writer, v int32) error {
	err := binary.Write(w, binary.LittleEndian, v)

	return err
}

// ReadSizeToFrame reads the sizes of lz4 blocks from a file and returns a
// frame.
func ReadSizeToFrame(r io.Reader) *Frame {
	f := new(Frame)

	if enc, err := ReadInt32(r); err != nil {
		log.Panicf("Unable to read encoded size: %s", err)
	} else {
		f.SizeCom = enc
	}

	if unc, err := ReadInt32(r); err != nil {
		log.Panicf("Unable to read unencoded size: %s", err)
	} else {
		f.SizeRaw = unc
	}

	f.isEncoded = true

	return f
}

// ReadJSONToFrame reads from a file into a Frame, compresses it, and sets the
// sizes.
func ReadJSONToFrame(fn string) *Frame {
	f := new(Frame)

	if r, err := os.Open(fn); err != nil {
		log.Panicf("Unable to open json file: %s", err)
	} else if n, err := io.Copy(f, r); err != nil {
		log.Panicf("Unable to read json file: %s", err)
	} else {
		f.SizeRaw = int32(n)
	}

	if err := f.Encode(); err != nil {
		log.Panicf("Unable to compress Frame: %s", err)
	}

	f.isEncoded = false

	return f
}

// CheckHeader checks the magic number and version number in the save file.
func CheckHeader(r io.Reader) {
	if m, err := ReadInt32(r); err != nil {
		log.Panicf("Failed magic number check: %s", err)
	} else if m != Magic {
		log.Panicf("Incorrect magic number: %d", m)
	}

	if v, err := ReadInt32(r); err != nil {
		log.Panicf("Failed version number check: %s", err)
	} else if v != Ver {
		log.Panicf("Incorrect version number: %x", v)
	}
}

// WriteJSON reads a file to a Frame, decodes it, and writes the decoded
// Frame to a file.
func WriteJSON(fn string, r io.Reader, f *Frame) {
	if _, err := io.CopyN(f, r, int64(f.SizeCom)); err != nil {
		log.Panicf("Unable to read file: %s", err)
	}

	if err := f.Decode(); err != nil {
		log.Panicf("Unable to decode: %s", err)
	}

	if err := ioutil.WriteFile(fn, f.Bytes(), 0644); err != nil {
		log.Panicf("Unable to write file: %s", err)
	}
}

// WriteHeader writes the magic number and version number to a save file.
func WriteHeader(w io.Writer) {
	if err := WriteInt32(w, Magic); err != nil {
		log.Panicf("Unable to write magic number: %s", err)
	}
	if err := WriteInt32(w, Ver); err != nil {
		log.Panicf("Unable to write version number: %s", err)
	}
}

// WriteSize writes size to a save file.
func WriteSize(w io.Writer, f *Frame) {
	if err := WriteInt32(w, f.SizeCom); err != nil {
		log.Panicf("Unable to write encoded size: %s", err)
	}

	if err := WriteInt32(w, f.SizeRaw); err != nil {
		log.Panicf("Unable to write unencoded size: %s", err)
	}
}

// WriteFrame writes the Frame to a save file.
func WriteFrame(w io.Writer, f *Frame) {
	if _, err := io.Copy(w, f); err != nil {
		log.Panicf("Unable to write Frame to save file: %s", err)
	}
}
