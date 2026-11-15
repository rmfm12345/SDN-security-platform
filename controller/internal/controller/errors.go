package controller

import (
	"encoding/binary"
	"fmt"
	"log"

	"antrea.io/libOpenflow/openflow15"
)

func (c *Controller) handleError(sc *switchConn, m *openflow15.ErrorMsg) {
	failed := "unknown"

	data := m.Data.Bytes()
	if len(data) >= 8 {
		failedType := data[1]
		failedXid := binary.BigEndian.Uint32(data[4:8])

		failed = fmt.Sprintf(
			"%s (xid: %d)",
			messageTypeName(failedType),
			failedXid,
		)
	}

	log.Printf(
		"ERROR from %s: dpid=%016x type=%s code=%s failed_request=%s",
		sc.addr,
		sc.dpid,
		errorTypeName(m.Type),
		errorCodeName(m.Type, m.Code),
		failed,
	)
}

var messageTypeNames = map[uint8]string{
	openflow15.Type_Hello:            "HELLO",
	openflow15.Type_Error:            "ERROR",
	openflow15.Type_EchoRequest:      "ECHO_REQUEST",
	openflow15.Type_EchoReply:        "ECHO_REPLY",
	openflow15.Type_FeaturesRequest:  "FEATURES_REQUEST",
	openflow15.Type_FeaturesReply:    "FEATURES_REPLY",
	openflow15.Type_PacketIn:         "PACKET_IN",
	openflow15.Type_PacketOut:        "PACKET_OUT",
	openflow15.Type_FlowMod:          "FLOW_MOD",
	openflow15.Type_MultiPartRequest: "MULTIPART_REQUEST",
	openflow15.Type_MultiPartReply:   "MULTIPART_REPLY",
	openflow15.Type_BarrierRequest:   "BARRIER_REQUEST",
}

var errorTypeNames = map[uint16]string{
	openflow15.ET_HELLO_FAILED:         "HELLO_FAILED",
	openflow15.ET_BAD_REQUEST:          "BAD_REQUEST",
	openflow15.ET_BAD_ACTION:           "BAD_ACTION",
	openflow15.ET_BAD_INSTRUCTION:      "BAD_INSTRUCTION",
	openflow15.PET_BAD_MATCH:           "BAD_MATCH",
	openflow15.ET_FLOW_MOD_FAILED:      "FLOW_MOD_FAILED",
	openflow15.ET_GROUP_MOD_FAILED:     "GROUP_MOD_FAILED",
	openflow15.ET_PORT_MOD_FAILED:      "PORT_MOD_FAILED",
	openflow15.ET_TABLE_MOD_FAILED:     "TABLE_MOD_FAILED",
	openflow15.ET_SWITCH_CONFIG_FAILED: "SWITCH_CONFIG_FAILED",
}

// errorCodeNames maps error type -> error code -> name.
// Only the types we are likely to hit for now are listed;
// add more as the controller grows.
var errorCodeNames = map[uint16]map[uint16]string{
	openflow15.ET_HELLO_FAILED: {
		openflow15.HFC_INCOMPATIBLE: "INCOMPATIBLE",
		openflow15.HFC_EPERM:        "EPERM",
	},
	openflow15.ET_BAD_REQUEST: {
		openflow15.BRC_BAD_VERSION:   "BAD_VERSION",
		openflow15.BRC_BAD_TYPE:      "BAD_TYPE",
		openflow15.BRC_BAD_MULTIPART: "BAD_MULTIPART",
		openflow15.BRC_EPERM:         "EPERM",
		openflow15.BRC_BAD_LEN:       "BAD_LEN",
		openflow15.BRC_BAD_TABLE_ID:  "BAD_TABLE_ID",
		openflow15.BRC_BAD_PORT:      "BAD_PORT",
		openflow15.BRC_BAD_PACKET:    "BAD_PACKET",
	},
}

func messageTypeName(t uint8) string {
	if name, ok := messageTypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("TYPE_%d", t)
}

func errorTypeName(t uint16) string {
	if name, ok := errorTypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("%d", t)
}

func errorCodeName(t, code uint16) string {
	if name, ok := errorCodeNames[t][code]; ok {
		return name
	}
	return fmt.Sprintf("%d", code)
}
