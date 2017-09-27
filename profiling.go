// +build profiling

package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
)

func init() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	runtime.SetMutexProfileFraction(1)
	runtime.SetBlockProfileRate(1)
}
