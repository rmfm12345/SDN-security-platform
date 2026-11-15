package controller

import (
	"log"

	"antrea.io/libOpenflow/openflow15"

	"github.com/rmfm12345/SDN_Network_Security/controller/internal/topology"
)

// handlePortStatus processes a PORT_STATUS message: the switch tells us
// that one of its ports was added, removed or changed (e.g. link down).
func (c *Controller) handlePortStatus(sc *switchConn, m *openflow15.PortStatus) {
	if m.Desc.PortNo == openflow15.P_LOCAL {
		return
	}

	p := toTopologyPort(&m.Desc)
	end := topology.LinkEnd{DPID: sc.dpid, Port: p.Number}

	switch m.Reason {
	case openflow15.PR_DELETE:
		c.registry.RemovePort(sc.dpid, p.Number)
		log.Printf("PORT DELETED: dpid=%016x port=%d name=%s", sc.dpid, p.Number, p.Name)
		c.removeLinks(end)

	case openflow15.PR_ADD, openflow15.PR_MODIFY:
		c.registry.AddPort(sc.dpid, p)
		log.Printf("PORT %s: dpid=%016x port=%d name=%s", p.State, sc.dpid, p.Number, p.Name)

		if p.State == "live" {
			// Rediscover links on this switch right away
			// instead of waiting for the next LLDP tick.
			c.sendLLDP(sc)
		} else {
			c.removeLinks(end)
		}
	}
}

// removeLinks drops every link that touches the given port.
func (c *Controller) removeLinks(end topology.LinkEnd) {
	for _, l := range c.links.RemovePort(end) {
		log.Printf(
			"LINK REMOVED: %016x:%d -> %016x:%d (port down)",
			l.Src.DPID, l.Src.Port,
			l.Dst.DPID, l.Dst.Port,
		)
	}
}
