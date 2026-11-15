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

// Ports returns a copy of the ports of a switch.
//
// It returns a copy, not the internal map, so the caller can use it
// without holding the lock (and without racing with AddPort).
func (r *SwitchRegistry) Ports(dpid uint64) []Port {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sw, ok := r.switches[dpid]
	if !ok {
		return nil
	}

	ports := make([]Port, 0, len(sw.Ports))
	for _, p := range sw.Ports {
		ports = append(ports, p)
	}
	return ports
}

// RemovePort deltes a port from a switch
func (r *SwitchRegistry) RemovePort(dpid uint64, portNo uint32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if sw, ok := r.switches[dpid]; ok {
		delete(sw.Ports, portNo)
	}
}
