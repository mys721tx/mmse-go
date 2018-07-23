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

type buff struct {
	unencoded []byte
	encoded   []byte
}

// read fills the encoded buffer
func (b *buff) read(r io.Reader) error {
	n, err := r.Read(b.encoded)

	if err != nil {
		return err
	}

	if s := len(b.encoded); s != n {
		return fmt.Errorf("Expecting %d bytes, read %d", s, n)
	}

	return nil
}

// decode decodes blocks in b.encoded to b.unencoded
func (b *buff) decode() error {
	si, err := lz4.UncompressBlock(b.encoded, b.unencoded)

	if err != nil {
		return err
	}

	if s := len(b.unencoded); s != si {
		return fmt.Errorf("Expecting %d bytes, got %d", s, si)
	}

	return nil
}

// readInt32 reads an int32 from a file.
func readInt32(r io.Reader) (int32, error) {
	var v int32

	err := binary.Read(r, binary.LittleEndian, &v)

	if err != nil {
		return 0, err
	}

	return v, nil
}

// newBuf returns a buffer for lz4 to use
func newBuf(r io.Reader) (*buff, error) {
	enc, err := readInt32(r)

	if err != nil {
		return nil, fmt.Errorf("Unable to read encoded size: %s", err)
	}

	unc, err := readInt32(r)

	if err != nil {
		return nil, fmt.Errorf("Unable to read unencoded size: %s", err)
	}

	return &buff{unencoded: make([]byte, unc), encoded: make([]byte, enc)}, nil
}

// checkMagic checks the magic number in the save file.
func checkMagic(r io.Reader) {
	m, err := readInt32(r)

	if err != nil {
		log.Fatalf("Failed magic number check: %s", err)
	}

	if m != magic {
		log.Fatalf("Incorrect magic number: %d", m)
	}
}

// checkVer checks the version number in the save file.
func checkVer(r io.Reader) {
	v, err := readInt32(r)

	if err != nil {
		log.Fatalf("Failed version number check: %s", err)
	}

	if v != ver {
		log.Fatalf("Failed version number check: %x", v)
	}
}

// writeJSON writes the buffer to a file.
func writeJSON(fn string, f io.Reader, b *buff) {
	err := b.read(f)
	if err != nil {
		log.Fatalf("Unable to read buffer: %s", err)
	}

	err = b.decode()
	if err != nil {
		log.Fatalf("Unable to decode buffer: %s", err)
	}

	err = ioutil.WriteFile(fn, b.unencoded, 0644)
	if err != nil {
		log.Fatalf("Unable to write file: %s", err)
	}
}

// unpack is a wrapper for unpacking json files.
func unpack(fn string) {
	bn := path.Base(fn)

	f, err := os.Open(fn)
	if err != nil {
		log.Fatalf("Unable to open %s: %s", fn, err)
	}

	defer func() {
		err = f.Close()
		if err != nil {
			log.Fatalf("Unable to close %s: %s", fn, err)
		}
	}()

	checkMagic(f)
	checkVer(f)

	info, err := newBuf(f)
	if err != nil {
		log.Fatalln(err)
	}

	data, err := newBuf(f)
	if err != nil {
		log.Fatalln(err)
	}

	writeJSON(fmt.Sprintf("%s_info.json", bn), f, info)
	writeJSON(fmt.Sprintf("%s_data.json", bn), f, data)
}

func main() {

	switch len(os.Args) {
	case 2:
		// unpack when parameos.Args has one file
		unpack(os.Args[1])
	case 3:
		// pack when os.Args has two files
		fmt.Println(os.Args[1:])
	default:
		// print usage in other case
		fmt.Printf(usg, os.Args[0])
	}
}
