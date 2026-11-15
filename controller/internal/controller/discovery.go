package controller

import (
	"log"
	"net"
	"time"

	"antrea.io/libOpenflow/openflow15"
	"antrea.io/libOpenflow/util"

	"github.com/rmfm12345/SDN_Network_Security/controller/internal/lldp"
	"github.com/rmfm12345/SDN_Network_Security/controller/internal/topology"
)

// How often each swithc floods LLDP out of all its ports.
const lldpInterval = 5 * time.Second

// Priority of the LLDP flow: the highest, so no forwarding flow
// installed later can swallow LLDP frames befroe they reach us.
const lldpFlowPriority = 0xffff

// instalLLDPFlow tells the switch to send every LLDP frame it
// receives to the controller.
func (c *Controller) installLLDPFlow(sc *switchConn) {
	flow := openflow15.NewFlowMod()
	flow.TableId = 0
	flow.Priority = lldpFlowPriority

	flow.Match.AddField(*openflow15.NewEthTypeField(lldp.EtherType))

	actions := openflow15.NewInstrApplyActions()
	if err := actions.AddAction(
		openflow15.NewActionOutput(openflow15.P_CONTROLLER),
		false,
	); err != nil {
		log.Printf("failed too build LLDP flow for %s: %v", sc.addr, err)
		return
	}
	flow.AddInstruction(actions)

	sc.stream.Outbound <- flow

	log.Printf("LLDP flow installed on dpid=%016x", sc.dpid)
}

// sendLLDP sends one LLDP frame out of every port of the switch
func (c *Controller) sendLLDP(sc *switchConn) {
	if sc.dpid == 0 {
		return
	}

	for _, port := range c.registry.Ports(sc.dpid) {
		mac, err := net.ParseMAC(port.HWAddr)
		if err != nil {
			log.Printf("bad MAC %q on dpid=%016x port=%d", port.Name, sc.dpid, port.Number)
			continue
		}

		frame := lldp.Build(sc.dpid, port.Number, mac)

		out := openflow15.NewPacketOut()

		out.Match.AddField(*openflow15.NewInPortField(openflow15.P_CONTROLLER))

		out.AddAction(openflow15.NewActionOutput(port.Number))
		out.Data = util.NewBuffer(frame)

		sc.stream.Outbound <- out
	}
}

// handleLLDP processes an LLDP frame received by a switch.
//
// The frame says where it was sent from (src); the PACKET_IN says
// where it arrived (dst). Together they describe one link.
func (c *Controller) handleLLDP(sc *switchConn, inPort uint32, frame []byte) {
	srcDPID, srcPort, err := lldp.Parse(frame)
	if err != nil {
		log.Printf("ignoring LLDP on dpid=%016x port=%d: %v", sc.dpid, inPort, err)
		return
	}

	src := topology.LinkEnd{DPID: srcDPID, Port: srcPort}
	dst := topology.LinkEnd{DPID: sc.dpid, Port: inPort}

	if c.links.Update(src, dst) {
		log.Printf(
			"LINK DISCOVERED: %016x:%d -> %016x:%d",
			src.DPID, src.Port,
			dst.DPID, dst.Port,
		)
	}
}
