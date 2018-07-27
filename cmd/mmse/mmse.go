/*
 *  mmso-go: Motorsport Manager save edit suite
 *  Copyright (C) 2018  Yishen Miao
 *
 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation, either version 3 of the License, or
 *  (at your option) any later version.
 *
 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 */

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/pierrec/lz4"
)

var (
	// magic and ver are little endian
	magic int32 = 0x73326d6d
	ver   int32 = 0x00000004
	usg         = `Usage: %[1]s	[game.sav]
Or:	%[1]s [info.json] [data.json]
`
)

type frame struct {
	sizeRaw int32
	sizeCom int32
	bytes.Buffer
}

// decode decodes the content in place.
func (f *frame) decode() error {
	b := make([]byte, f.sizeRaw)

	n, err := lz4.UncompressBlock(f.Bytes(), b)

	if err != nil {
		return err
	}

	if int32(n) != f.sizeRaw {
		return fmt.Errorf(
			"Expecting %d bytes, read %d",
			f.sizeRaw, int32(n),
		)
	}

	f.Reset()

	_, err = f.Write(b)

	return err
}

func (f *frame) encode() error {
	b := make([]byte, f.sizeRaw)

	n, err := lz4.CompressBlock(f.Bytes(), b, make([]int, 1<<16))

	if err != nil {
		return err
	}

	// lz4.CompressBLock returns 0 if the data is not compressible.
	if n == 0 {
		f.sizeCom = f.sizeRaw
	} else {
		f.sizeCom = int32(n)
	}

	f.Reset()

	_, err = f.Write(b)

	f.Truncate(int(f.sizeCom))

	return err
}

// readInt32 reads an int32 from a file.
func readInt32(r io.Reader) (int32, error) {
	var v int32

	if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
		return 0, err
	}

	return v, nil
}

// writeInt32 writes an int32 from a file.
func writeInt32(w io.Writer, v int32) error {

	err := binary.Write(w, binary.LittleEndian, v)

	return err
}

// readSizeToFrame reads the sizes of lz4 blocks from a file and returns a
// frame.
func readSizeToFrame(r io.Reader) *frame {
	f := new(frame)

	if enc, err := readInt32(r); err != nil {
		log.Panicf("Unable to read encoded size: %s", err)
	} else {
		f.sizeCom = enc
	}

	if unc, err := readInt32(r); err != nil {
		log.Panicf("Unable to read unencoded size: %s", err)
	} else {
		f.sizeRaw = unc
	}

	return f
}

// readJSONToFrame reads from a file into a frame, compresses it, and sets the
// sizes.
func readJSONToFrame(fn string) *frame {
	f := new(frame)

	if r, err := os.Open(fn); err != nil {
		log.Panicf("Unable to open json file: %s", err)
	} else if n, err := io.Copy(f, r); err != nil {
		log.Panicf("Unable to read json file: %s", err)
	} else {
		f.sizeRaw = int32(n)
	}

	if err := f.encode(); err != nil {
		log.Panicf("Unable to compress frame: %s", err)
	}

	return f
}

// checkHeader checks the magic number and version number in the save file.
func checkHeader(r io.Reader) {
	if m, err := readInt32(r); err != nil {
		log.Panicf("Failed magic number check: %s", err)
	} else if m != magic {
		log.Panicf("Incorrect magic number: %d", m)
	}

	if v, err := readInt32(r); err != nil {
		log.Panicf("Failed version number check: %s", err)
	} else if v != ver {
		log.Panicf("Incorrect version number: %x", v)
	}
}

// writeJSON reads a file to a frame, decodes it, and writes the decoded
// frame to a file.
func writeJSON(fn string, r io.Reader, f *frame) {
	if _, err := io.CopyN(f, r, int64(f.sizeCom)); err != nil {
		log.Panicf("Unable to read file: %s", err)
	}

	if err := f.decode(); err != nil {
		log.Panicf("Unable to decode: %s", err)
	}

	if err := ioutil.WriteFile(fn, f.Bytes(), 0644); err != nil {
		log.Panicf("Unable to write file: %s", err)
	}
}

// writeHeader writes the magic number and version number to a save file.
func writeHeader(w io.Writer) {
	if err := writeInt32(w, magic); err != nil {
		log.Panicf("Unable to write magic number: %s", err)
	}
	if err := writeInt32(w, ver); err != nil {
		log.Panicf("Unable to write version number: %s", err)
	}
}

// writeSize writes size to a save file.
func writeSize(w io.Writer, f *frame) {
	if err := writeInt32(w, f.sizeCom); err != nil {
		log.Panicf("Unable to write encoded size: %s", err)
	}

	if err := writeInt32(w, f.sizeRaw); err != nil {
		log.Panicf("Unable to write unencoded size: %s", err)
	}
}

// writeFrame writes the frame to a save file.
func writeFrame(w io.Writer, f *frame) {
	if _, err := io.Copy(w, f); err != nil {
		log.Panicf("Unable to write frame to save file: %s", err)
	}
}

// split splits a file name into base and extension. Modified from path.Ext().
func split(fn string) string {
	for i := len(fn) - 1; i >= 0; i-- {
		if fn[i] == '.' {
			if fn[i:] == ".sav" || fn[i:] == ".json" {
				return fn[:i]
			}
			break
		}
	}
	return fn
}

// unpack is a wrapper for unpacking json files.
func unpack(fn string) {
	bn := split(path.Base(fn))

	f, err := os.Open(fn)
	if err != nil {
		log.Panicf("Unable to open %s: %s", fn, err)
	}

	defer func() {
		err = f.Close()
		if err != nil {
			log.Panicf("Unable to close %s: %s", fn, err)
		}
	}()

	checkHeader(f)

	info := readSizeToFrame(f)

	data := readSizeToFrame(f)

	writeJSON(fmt.Sprintf("%s_info.json", bn), f, info)
	writeJSON(fmt.Sprintf("%s_data.json", bn), f, data)
}

// unpack is a wrapper for packing json files.
func pack(in, dn string) {
	bn := split(path.Base(dn))

	f, err := os.Create(fmt.Sprintf("%s.sav", bn))

	if err != nil {
		log.Panicf("%s", err)
	}

	defer func() {
		if err = f.Close(); err != nil {
			log.Panicf("Unable to close file: %s", err)
		}
	}()

	writeHeader(f)

	info := readJSONToFrame(in)

	writeSize(f, info)

	data := readJSONToFrame(dn)

	writeSize(f, data)

	writeFrame(f, info)
	writeFrame(f, data)
}

func main() {

	switch len(os.Args) {
	case 2:
		// unpack when parameos.Args has one file
		unpack(os.Args[1])
	case 3:
		// pack when os.Args has two files
		pack(os.Args[1], os.Args[2])
	default:
		// print usage in other case
		fmt.Printf(usg, os.Args[0])
	}
}
