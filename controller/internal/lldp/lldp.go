// Package lldp builds and parses the minimal LLDP frames (IEEE 802.1AB)
// the controller uses for link discovery.
//
// Frame layout:
//
//      Ethernet: dst 01:80:c2:00:00:0e | src | EtherType 0x88cc
//      TLV Chassis ID  (type 1): subtype 7 (locally assigned), "dpid:<16 hex>"
//      TLV Port ID     (type 2): subtype 2 (port component), 4-byte port number
//      TLV TTL         (type 3): 2-byte seconds
//      TLV End         (type 0): empty
//
// Each TLV starts with a 2-byte header: 7 bits of type + 9 bits of length.

package lldp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	EtherType = 0x88cc

	tlvEnd       = 0
	tlvChassisID = 1
	tlvPortID    = 2
	tlvTTL       = 3

	chassisSubtypeLocal  = 7
	portSubtypeComponent = 2

	chassisPrefix = "dpid:"
	defaultTTL    = 120
)

// MulticastMAC is the "nearest bridge" LLDP address. Bridges never
// forward frames sent to it, so an LLDP frame only crosses one link.
var MulticastMAC = net.HardwareAddr{0x01, 0x80, 0xc2, 0x00, 0x00, 0x0e}

func Build(dpid uint64, port uint32, srcMAC net.HardwareAddr) []byte {
	frame := make([]byte, 0, 64)

	// Ethernet header.
	frame = append(frame, MulticastMAC...)
	frame = append(frame, srcMAC...)
	frame = binary.BigEndian.AppendUint16(frame, EtherType)

	chassis := append([]byte{chassisSubtypeLocal}, fmt.Sprintf("%s%016x", chassisPrefix, dpid)...)
	frame = appendTLV(frame, tlvChassisID, chassis)

	portID := binary.BigEndian.AppendUint32([]byte{portSubtypeComponent}, port)
	frame = appendTLV(frame, tlvPortID, portID)

	frame = appendTLV(frame, tlvTTL, binary.BigEndian.AppendUint16(nil, defaultTTL))

	frame = appendTLV(frame, tlvEnd, nil)

	return frame
}

func Parse(frame []byte) (dpid uint64, port uint32, err error) {
	if len(frame) < 14 {
		return 0, 0, errors.New("frame too short")
	}
	if binary.BigEndian.Uint16(frame[12:14]) != EtherType {
		return 0, 0, errors.New("not an LLDP frame")
	}

	var haveChassis, havePort bool
	data := frame[14:]

	for len(data) >= 2 {
		header := binary.BigEndian.Uint16(data[:2])
		tlvType := header >> 9
		tlvLen := int(header & 0x01ff)
		data = data[2:]

		if tlvType == tlvEnd {
			break
		}
		if len(data) < tlvLen {
			return 0, 0, errors.New("truncated TLV")
		}
		value := data[:tlvLen]
		data = data[tlvLen:]

		switch tlvType {
		case tlvChassisID:
			if len(value) < 1 || value[0] != chassisSubtypeLocal {
				return 0, 0, errors.New("unexpected chassis ID subtype")
			}
			id := string(value[1:])
			if !strings.HasPrefix(id, chassisPrefix) {
				return 0, 0, fmt.Errorf("chassis ID %q is not ours", id)
			}
			dpid, err = strconv.ParseUint(strings.TrimPrefix(id, chassisPrefix), 16, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("bad dpid in chassis ID: %w", err)
			}
			haveChassis = true

		case tlvPortID:
			if len(value) != 5 || value[0] != portSubtypeComponent {
				return 0, 0, errors.New("unexpected port ID format")
			}
			port = binary.BigEndian.Uint32(value[1:])
			havePort = true
		}
	}
	if !haveChassis || !havePort {
		return 0, 0, errors.New("missing chassis or port TLV")
	}
	return dpid, port, nil
}

func appendTLV(b []byte, tlvType uint16, value []byte) []byte {
	header := tlvType<<9 | uint16(len(value))
	b = binary.BigEndian.AppendUint16(b, header)
	return append(b, value...)
}
