package controller

import (
	"log"
	"sync"

	of "github.com/netrack/openflow"
	"github.com/netrack/openflow/ofp"

	"github.com/rmfm12345/SDN_Network_Security/controller/internal/topology"
)

const listenAddress = ":6653"

type Controller struct {
	registry *topology.SwitchRegistry

	mu          sync.RWMutex
	connections map[string]uint64
}

func New() *Controller {
	return &Controller{
		registry:    topology.NewSwitchRegistry(),
		connections: make(map[string]uint64),
	}
}

func (c *Controller) Start() {
	of.HandleFunc(of.TypeHello, c.handleHello)
	of.HandleFunc(of.TypeFeaturesReply, c.handleFeaturesReply)
	of.HandleFunc(of.TypeMultipartReply, c.handleMultipartReply)
	of.HandleFunc(of.TypeEchoRequest, c.handleEchoRequest)

	log.Printf("Starting SDN controller on %s", listenAddress)

	if err := of.ListenAndServe(listenAddress, nil); err != nil {
		log.Fatalf("controller stopped: %v", err)
	}
}

func (c *Controller) handleHello(
	rw of.ResponseWriter,
	r *of.Request,
) {
	log.Printf("HELLO received from %s", r.Addr)

	err := rw.Write(&of.Header{
		Version:     r.Header.Version,
		Type:        of.TypeHello,
		Transaction: r.Header.Transaction,
	}, nil)

	if err != nil {
		log.Printf("failed to send HELLO: %v", err)
		return
	}

	request := of.NewRequest(of.TypeFeaturesRequest, nil)

	if err := r.Conn().Send(request); err != nil {
		log.Printf(
			"failed to send FEATURES_REQUEST to %s: %v",
			r.Addr,
			err,
		)
		return
	}

	if err := r.Conn().Flush(); err != nil {
		log.Printf(
			"failed to flush FEATURES_REQUEST to %s: %v",
			r.Addr,
			err,
		)
		return
	}

	log.Printf("FEATURES_REQUEST sent to %s", r.Addr)
}

func (c *Controller) handleFeaturesReply(
	rw of.ResponseWriter,
	r *of.Request,
) {
	var features ofp.SwitchFeatures

	if _, err := features.ReadFrom(r.Body); err != nil {
		log.Printf(
			"failed to read FEATURES_REPLY from %s: %v",
			r.Addr,
			err,
		)
		return
	}

	c.mu.Lock()
	c.connections[r.Addr.String()] = features.DatapathID
	c.mu.Unlock()

	sw := &topology.Switch{
		DPID:  features.DatapathID,
		Ports: make(map[uint32]topology.Port),
	}

	c.registry.AddSwitch(sw)

	log.Printf(
		"SWITCH CONNECTED: addr=%s dpid=%016x tables=%d buffers=%d",
		r.Addr,
		features.DatapathID,
		features.NumTables,
		features.NumBuffers,
	)

	c.requestPortDescriptions(r)
}

func (c *Controller) requestPortDescriptions(r *of.Request) {
	body := ofp.NewMultipartRequest(
		ofp.MultipartTypePortDescription,
		nil,
	)

	request := of.NewRequest(
		of.TypeMultipartRequest,
		body,
	)

	if err := r.Conn().Send(request); err != nil {
		log.Printf(
			"failed to send PORT_DESC_REQUEST to %s: %v",
			r.Addr,
			err,
		)
		return
	}

	if err := r.Conn().Flush(); err != nil {
		log.Printf(
			"failed to flush PORT_DESC_REQUEST to %s: %v",
			r.Addr,
			err,
		)
		return
	}

	log.Printf(
		"PORT_DESC_REQUEST sent to %s",
		r.Addr,
	)
}

func (c *Controller) handleEchoRequest(
	rw of.ResponseWriter,
	r *of.Request,
) {
	var req ofp.EchoRequest

	if _, err := req.ReadFrom(r.Body); err != nil {
		log.Printf(
			"failed to read ECHO_REQUEST from %s: %v",
			r.Addr,
			err,
		)
		return
	}

	header := r.Header.Copy()
	header.Type = of.TypeEchoReply

	if err := rw.Write(header, &ofp.EchoReply{Data: req.Data}); err != nil {
		log.Printf(
			"failed to send ECHO_REPLY to %s: %v",
			r.Addr,
			err,
		)
	}
}

func (c *Controller) handleMultipartReply(
	rw of.ResponseWriter,
	r *of.Request,
) {
	var reply ofp.MultipartReply

	if _, err := reply.ReadFrom(r.Body); err != nil {
		log.Printf(
			"failed to read MULTIPART_REPLY from %s: %v",
			r.Addr,
			err,
		)
		return
	}

	if reply.Type != ofp.MultipartTypePortDescription {
		log.Printf(
			"received MULTIPART_REPLY type=%s from %s",
			reply.Type,
			r.Addr,
		)
		return
	}

	var ports ofp.Ports

	if _, err := ports.ReadFrom(r.Body); err != nil {
		log.Printf(
			"failed to read PORT_DESC_REPLY from %s: %v",
			r.Addr,
			err,
		)
		return
	}

	dpid, ok := c.findSwitchDPID(r.Addr.String())
	if !ok {
		log.Printf(
			"received PORT_DESC_REPLY from unknown switch %s",
			r.Addr,
		)
		return
	}

	log.Printf(
		"PORT_DESC_REPLY received from %s: dpid=%016x ports=%d",
		r.Addr,
		dpid,
		len(ports),
	)

	for _, port := range ports {
		// OFPP_LOCAL is a reserved pseudo-port,
		// not a physical switch interface.
		if port.PortNo == 0xfffffffe {
			continue
		}

		p := topology.Port{
			Number: uint32(port.PortNo),
			Name:   port.Name,
			HWAddr: port.HWAddr.String(),
			State:  port.State.String(),
		}

		if !c.registry.AddPort(dpid, p) {
			log.Printf(
				"failed to add port %d to switch %016x",
				port.PortNo,
				dpid,
			)
			continue
		}

		log.Printf(
			"  port=%d name=%s hwaddr=%s state=%s",
			p.Number,
			p.Name,
			p.HWAddr,
			p.State,
		)
	}
}

func (c *Controller) findSwitchDPID(addr string) (uint64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	dpid, ok := c.connections[addr]
	return dpid, ok
}
