// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pprof

import (
	"net/http"
	"net/http/pprof"
)

func newHandler() http.Handler {
	handler := http.NewServeMux()
	handler.HandleFunc("/debug/pprof/", pprof.Index)
	handler.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	handler.HandleFunc("/debug/pprof/profile", pprof.Profile)
	handler.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	handler.HandleFunc("/debug/pprof/trace", pprof.Trace)
	handler.Handle("/debug/pprof/block", pprof.Handler("block"))
	handler.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	handler.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	handler.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	return handler
}
