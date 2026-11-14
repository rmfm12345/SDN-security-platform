package topology

type SwitchRegistry struct {
	switches map[uint64]*Switch
}

func NewSwitchRegistry() *SwitchRegistry {
	return &SwitchRegistry{
		switches: make(map[uint64]*Switch),
	}
}

func (r *SwitchRegistry) AddSwitch(sw *Switch) {
	r.switches[sw.DPID] = sw
}

func (r *SwitchRegistry) GetSwitch(dpid uint64) (*Switch, bool) {
	sw, ok := r.switches[dpid]
	return sw, ok
}

func (r *SwitchRegistry) AddPort(dpid uint64, port Port) bool {
	sw, ok := r.GetSwitch(dpid)
	if !ok {
		return false
	}

	sw.Ports[port.Number] = port
	return true
}
