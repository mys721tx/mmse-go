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

// readInt32 reads an int32 from a file.
func readInt32(r io.Reader) (int32, error) {
	var v int32

	err := binary.Read(r, binary.LittleEndian, &v)

	if err != nil {
		return 0, err
	}

	return v, nil
}

// readToFrame returns a buffer for lz4 to use
func readToFrame(r io.Reader) (*frame, error) {
	enc, err := readInt32(r)

	if err != nil {
		return nil, fmt.Errorf("Unable to read encoded size: %s", err)
	}

	unc, err := readInt32(r)

	if err != nil {
		return nil, fmt.Errorf("Unable to read unencoded size: %s", err)
	}

	return &frame{sizeRaw: unc, sizeCom: enc}, nil
}

// checkMagic checks the magic number in the save file.
func checkMagic(r io.Reader) {
	m, err := readInt32(r)

	if err != nil {
		log.Panicf("Failed magic number check: %s", err)
	}

	if m != magic {
		log.Panicf("Incorrect magic number: %d", m)
	}
}

// checkVer checks the version number in the save file.
func checkVer(r io.Reader) {
	v, err := readInt32(r)

	if err != nil {
		log.Panicf("Failed version number check: %s", err)
	}

	if v != ver {
		log.Panicf("Failed version number check: %x", v)
	}
}

// writeJSON writes the buffer to a file.
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

// unpack is a wrapper for unpacking json files.
func unpack(fn string) {
	bn := path.Base(fn)

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

	checkMagic(f)
	checkVer(f)

	info, err := readToFrame(f)
	if err != nil {
		log.Fatalln(err)
	}

	data, err := readToFrame(f)
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
