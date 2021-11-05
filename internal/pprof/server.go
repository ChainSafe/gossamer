package pprof

import (
	"net/http"
	"net/http/pprof"

	"github.com/ChainSafe/gossamer/internal/httpserver"
)

// NewServer creates a new Pprof server which will listen at
// the address specified.
func NewServer(address string, logger httpserver.Logger) *httpserver.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/debug/pprof/", pprof.Index)
	handler.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	handler.HandleFunc("/debug/pprof/profile", pprof.Profile)
	handler.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	handler.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return httpserver.New("pprof", address, handler, logger)
}
