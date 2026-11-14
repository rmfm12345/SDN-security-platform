package topology

type Port struct {
	Number uint32
	Name   string
	HWAddr string
	State  string
}

type Switch struct {
	DPID  uint64
	Ports map[uint32]Port
}
