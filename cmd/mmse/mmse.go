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

package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/mys721tx/mmse-go/pkg/mmse"
)

var (
	usg = `Usage: %[1]s	[game.sav]
Or:	%[1]s [info.json] [data.json]
`
)

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

	mmse.CheckHeader(f)

	info := mmse.ReadSizeToFrame(f)

	data := mmse.ReadSizeToFrame(f)

	mmse.WriteJSON(fmt.Sprintf("%s_info.json", bn), f, info)
	mmse.WriteJSON(fmt.Sprintf("%s_data.json", bn), f, data)
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

	mmse.WriteHeader(f)

	info := mmse.ReadJSONToFrame(in)

	mmse.WriteSize(f, info)

	data := mmse.ReadJSONToFrame(dn)

	mmse.WriteSize(f, data)

	mmse.WriteFrame(f, info)
	mmse.WriteFrame(f, data)
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
