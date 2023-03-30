package sync

import "time"

type WarpSync struct {
	network interface{}
}

func (w *WarpSync) sync() {
	w.waitForConnections()

}

func (w *WarpSync) waitForConnections() {
	// TODO: implement actual code to wait
	// for the minimal amount of peers
	time.Sleep(30 * time.Second)
}
