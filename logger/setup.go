// Copyright (C) 2025-present ObjectWeaver.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the Server Side Public License, version 1,
// as published by ObjectWeaver.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// Server Side Public License for more details.
//
// You should have received a copy of the Server Side Public License
// along with this program. If not, see
// <https://objectweaver.dev/licensing/server-side-public-license>.
package logger

import (
	"log"
	"os"
	"strconv"
)

type Out struct {
	verbose bool
}

var Output *Out

func init(){
	Output = &Out{}
	verbose, err := strconv.ParseBool(os.Getenv("VERBOSE"))
	if err != nil {
		verbose = false
	}

	Output.verbose = verbose
}

func (o *Out) Println(val any) {
	if (!o.verbose) {
		return
	}
	log.Println(val)
}