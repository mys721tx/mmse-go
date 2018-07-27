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

/*
mmse packs and unpacks the save file from Motorsport Manager.

When given one parameter, mmse unpacks the save file to an info JSON file and a
data JSON file. The JSON files use the file name of the save file as prefix.

When given two parameters, mmse packs the info JSON file and the data JSON file
to a save file. The save files use the file name of the data JSON file as
prefix.

Usage:
	mmse <savefile>
	mmse <infofile> <datafile>

*/
package main
