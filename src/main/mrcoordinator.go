package main

//
// start the coordinator process, which is implemented
// in ../mr/coordinator.go
//
// go run mrcoordinator.go pg*.txt
//
// Please do not change this file.
//

import (
	"fmt"
	"os"
	"time"

	"6.5840/mr"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: mrcoordinator inputfiles...\n")
		os.Exit(1)
	}

	// os.Args[1:] = [/home/verzil/coding/project/6.5840/src/main/pg-*.txt]
	m := mr.MakeCoordinator(os.Args[1:], 10)
	// fmt.Println(os.Args[1:])
	for m.Done() == false {
		time.Sleep(time.Second)
	}

	time.Sleep(time.Second)
}
