package topology

import "testing"

func TestRemoveSwitchKeepsNewerConnection(t *testing.T) {
	r := NewSwitchRegistry()

	oldSw := &Switch{DPID: 2, Ports: map[uint32]Port{}}
	newSw := &Switch{DPID: 2, Ports: map[uint32]Port{}}

	r.AddSwitch(oldSw)
	r.AddSwitch(newSw) // the switch reconnected before the old one was cleaned up

	if r.RemoveSwitch(oldSw) {
		t.Fatal("removed the switch of the newer connection")
	}
	if got, ok := r.GetSwitch(2); !ok || got != newSw {
		t.Fatal("newer connection is no longer registered")
	}
	if !r.RemoveSwitch(newSw) {
		t.Fatal("failed to remove the current switch")
	}
}
