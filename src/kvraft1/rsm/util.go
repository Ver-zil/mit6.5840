package rsm

import (
	"log"
	"os"
)

// Debugging
const Debug = false

var EnvDebug = os.Getenv("DEBUG") != ""

func DPrintf(format string, a ...interface{}) {
	if Debug || EnvDebug {
		log.Printf(format, a...)
	}
}