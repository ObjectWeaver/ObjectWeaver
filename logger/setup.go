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