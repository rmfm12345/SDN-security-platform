package topology

import "sync"

type SwitchRegistry struct {
	mu       sync.RWMutex
	switches map[uint64]*Switch
}

func NewSwitchRegistry() *SwitchRegistry {
	return &SwitchRegistry{
		switches: make(map[uint64]*Switch),
	}
}

func (r *SwitchRegistry) AddSwitch(sw *Switch) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.switches[sw.DPID] = sw
}

func (r *SwitchRegistry) GetSwitch(dpid uint64) (*Switch, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sw, ok := r.switches[dpid]
	return sw, ok
}

func (r *SwitchRegistry) AddPort(dpid uint64, port Port) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	sw, ok := r.switches[dpid]
	if !ok {
		return false
	}

	sw.Ports[port.Number] = port
	return true
}
