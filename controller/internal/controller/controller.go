package controller

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"antrea.io/libOpenflow/common"
	"antrea.io/libOpenflow/openflow15"
	"antrea.io/libOpenflow/protocol"
	"antrea.io/libOpenflow/util"

	"github.com/rmfm12345/SDN_Network_Security/controller/internal/lldp"
	"github.com/rmfm12345/SDN_Network_Security/controller/internal/topology"
)

const listenAddress = ":6653"

type Controller struct {
	registry *topology.SwitchRegistry
	links    *topology.LinkRegistry
}

func New() *Controller {
	return &Controller{
		registry: topology.NewSwitchRegistry(),
		links:    topology.NewLinkRegistry(),
	}
}

type parser struct{}

func (parser) Parse(b []byte) (util.Message, error) {
	return openflow15.Parse(b)
}

type switchConn struct {
	addr   string
	stream *util.MessageStream
	dpid   uint64
}

func (c *Controller) Start() {
	ln, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", listenAddress, err)
	}

	log.Printf("Starting SDN controller on %s", listenAddress)

	go c.expireLinks()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("failed to accept connection: %v", err)
			continue
		}

		go c.handleConnection(conn)
	}
}

func (c *Controller) handleConnection(conn net.Conn) {
	sc := &switchConn{
		addr:   conn.RemoteAddr().String(),
		stream: util.NewMessageStream(conn, parser{}),
	}

	log.Printf("TCP connection from %s", sc.addr)

	hello, err := common.NewHello(openflow15.VERSION)
	if err != nil {
		log.Printf("failed to build HELLO for %s: %v", sc.addr, err)
		return
	}
	sc.stream.Outbound <- hello

	lldpTicker := time.NewTicker(lldpInterval)
	defer lldpTicker.Stop()

	for {
		select {
		case msg := <-sc.stream.Inbound:
			c.dispatch(sc, msg)

		case <-lldpTicker.C:
			c.sendLLDP(sc)

		case err := <-sc.stream.Error:
			log.Printf(
				"SWITCH DISCONNECTED: addr=%s, dpid=%d, error=%v",
				sc.addr,
				sc.dpid,
				err,
			)
			return
		}
	}
}

func (c *Controller) dispatch(sc *switchConn, msg util.Message) {
	switch m := msg.(type) {
	case *common.Hello:
		c.handleHello(sc, m)

	case *openflow15.ErrorMsg:
		c.handleError(sc, m)

	case *common.Header:
		if m.Type == openflow15.Type_EchoRequest {
			c.handleEchoRequest(sc, m)
		}

	case *openflow15.SwitchFeatures:
		c.handleFeaturesReply(sc, m)

	case *openflow15.MultipartReply:
		c.handleMultipartReply(sc, m)

	case *openflow15.PacketIn:
		c.handlePacketIn(sc, m)

	case *openflow15.PortStatus:
		c.handlePortStatus(sc, m)

	default:
		log.Printf("unhandled message %T from %s", msg, sc.addr)
	}
}

func (c *Controller) handleHello(sc *switchConn, m *common.Hello) {
	log.Printf("HELLO received from %s : version=%d",
		sc.addr,
		m.Version,
	)

	sc.stream.Outbound <- openflow15.NewFeaturesRequest()

	log.Printf("FEATURES_REQUEST sent to %s", sc.addr)
}

func (c *Controller) handleEchoRequest(sc *switchConn, m *common.Header) {
	reply := openflow15.NewEchoReply()
	reply.Xid = m.Xid
	sc.stream.Outbound <- reply
}

func (c *Controller) handleFeaturesReply(
	sc *switchConn,
	m *openflow15.SwitchFeatures,
) {
	sc.dpid = binary.BigEndian.Uint64(m.DPID)

	c.registry.AddSwitch(&topology.Switch{
		DPID:  sc.dpid,
		Ports: make(map[uint32]topology.Port),
	})

	log.Printf(
		"SWITCH CONNECTED: addr=%s dpid=%016x tables=%d buffers=%d",
		sc.addr,
		sc.dpid,
		m.NumTables,
		m.Buffers,
	)

	c.requestPortDescriptions(sc)
	c.installTableMissFlow(sc)
	c.installLLDPFlow(sc)
}

func (c *Controller) requestPortDescriptions(sc *switchConn) {
	req := openflow15.NewMpRequest(openflow15.MultipartType_PortDesc)

	req.Body = append(
		req.Body,
		openflow15.NewPortMultipartRequest(openflow15.P_ANY),
	)

	sc.stream.Outbound <- req

	log.Printf("PORT_DESC_REQUEST sent to %s", sc.addr)
}

func (c *Controller) handleMultipartReply(
	sc *switchConn,
	m *openflow15.MultipartReply,
) {
	if m.Type != openflow15.MultipartType_PortDesc {
		log.Printf(
			"received MULTIPART_REPLY type=%d from %s",
			m.Type,
			sc.addr,
		)
		return
	}

	log.Printf(
		"PORT_DESC_REPLY received from %s: dpid=%016x, ports=%d",
		sc.addr,
		sc.dpid,
		len(m.Body),
	)

	for _, body := range m.Body {
		port, ok := body.(*openflow15.Port)
		if !ok {
			continue
		}

		if port.PortNo == openflow15.P_LOCAL {
			continue
		}

		p := toTopologyPort(port)

		if !c.registry.AddPort(sc.dpid, p) {
			log.Printf(
				"failed to add port %d to switch %016x",
				p.Number,
				sc.dpid,
			)
			continue
		}

		log.Printf(
			" port=%d name=%s hwaddr=%s state=%s",
			p.Number,
			p.Name,
			p.HWAddr,
			p.State,
		)
	}
	c.sendLLDP(sc)
}

// toTopologyPort converts an OpenFlow port description into our Port.
func toTopologyPort(port *openflow15.Port) topology.Port {
	return topology.Port{
		Number: port.PortNo,
		Name:   strings.TrimRight(string(port.Name), "\x00"),
		HWAddr: port.HWAddr.String(),
		State:  portStateString(port.State),
	}
}

func portStateString(state uint32) string {
	switch {
	case state&openflow15.PS_LINK_DOWN != 0:
		return "down"
	case state&openflow15.PS_BLOCKED != 0:
		return "blocked"
	case state&openflow15.PS_LIVE != 0:
		return "live"
	default:
		return "unknown"
	}
}

// installTableMissFlow installs the lowest-priority flow, which matches
// every packet that no other flow matches and sends it to the controller.
func (c *Controller) installTableMissFlow(sc *switchConn) {
	flow := openflow15.NewFlowMod()
	flow.TableId = 0
	flow.Priority = 0

	actions := openflow15.NewInstrApplyActions()
	if err := actions.AddAction(
		openflow15.NewActionOutput(openflow15.P_CONTROLLER),
		false,
	); err != nil {
		log.Printf("failed to build table-miss flow for %s: %v", sc.addr, err)
		return
	}
	flow.AddInstruction(actions)

	sc.stream.Outbound <- flow

	log.Printf("TABLE-MISS flow installed on dpid=%016x", sc.dpid)
}

func (c *Controller) handlePacketIn(sc *switchConn, m *openflow15.PacketIn) {
	inPort, ok := packetInPort(m)
	if !ok {
		log.Printf("PACKET_IN from %s without in_port", sc.addr)
		return
	}

	raw, err := m.Data.MarshalBinary()
	if err != nil {
		log.Printf("failed to read PACKET_IN data from %s: %v", sc.addr, err)
		return
	}

	if len(raw) >= 14 && binary.BigEndian.Uint16(raw[12:14]) == lldp.EtherType {
		c.handleLLDP(sc, inPort, raw)
		return
	}

	eth := new(protocol.Ethernet)
	if err := eth.UnmarshalBinary(raw); err != nil {
		log.Printf("failed to parse Ethernet frame from %s: %v", sc.addr, err)
		return
	}

	log.Printf(
		"PACKET_IN dpid=%016x in_port=%d src=%s dst=%s type=%s",
		sc.dpid,
		inPort,
		eth.HWSrc,
		eth.HWDst,
		etherTypeName(eth.Ethertype),
	)
}

func packetInPort(m *openflow15.PacketIn) (uint32, bool) {
	for _, field := range m.Match.Fields {
		if field.Class != openflow15.OXM_CLASS_OPENFLOW_BASIC ||
			field.Field != openflow15.OXM_FIELD_IN_PORT {
			continue
		}

		if v, ok := field.Value.(*openflow15.InPortField); ok {
			return v.InPort, true
		}
	}
	return 0, false
}

func etherTypeName(t uint16) string {
	switch t {
	case protocol.IPv4_MSG:
		return "IPv4"
	case protocol.IPv6_MSG:
		return "IPv6"
	case protocol.ARP_MSG:
		return "ARP"
	case protocol.LLDP_MSG:
		return "LLDP"
	default:
		return fmt.Sprintf("0x%04x", t)
	}
}
