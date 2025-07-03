package raft

import (
	"log"

	tester "6.5840/tester1"
)

// Debugging
const Debug = true

func DPrintf(format string, a ...interface{}) {
	if Debug {
		log.Printf(format, a...)
	}
}

func Record(tag, desp, details string) {
	// 这个函数将DPrint和tester.Annotation集成
	tester.Annotate(tag, desp, details)
	DPrintf(details)
}
