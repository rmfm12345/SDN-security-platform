# SDN Network Security

An SDN controller written from scratch in **Go**, focused on network security, and tested on an emulated network built with **Mininet** and **Open vSwitch** over **OpenFlow 1.5**.

OpenFlow message encoding/decoding is handled by [`antrea.io/libOpenflow`](https://github.com/antrea-io/libOpenflow) (`openflow15` package). The TCP server, per-switch connection handling and message dispatch are implemented in this project.

## Architecture

**Control plane:** every switch connects to the controller at `tcp:127.0.0.1:6653` (OpenFlow 1.5).

**Data plane:**
```
           ┌────────────────────────┐
           │                        │
  h1 ──── s1 ──────── s2 ──────── s3 ──── h3
                      │
                      h2
```

- 3 hosts (`h1`, `h2`, `h3`), each connected to one switch.
- 3 OVS switches connected in a triangle (`s1–s2`, `s2–s3`, `s1–s3`), which creates a loop in the topology.
- The switches run in `fail-mode secure`: they only forward traffic according to the flows installed by the controller.

### Controller design

```
Start()  ── net.Listen(":6653") ── Accept() loop
              │
              ├── go handleConnection(conn)     one goroutine per switch
              │      ├── util.MessageStream      (libOpenflow: decode/encode)
              │      ├── sends HELLO
              │      └── for/select:
              │            Inbound    → dispatch(msg) → handleHello / handleFeaturesReply /
              │                                          handleMultipartReply / handleEchoRequest /
              │                                          handlePacketIn / handleError
              │            LLDP ticker → sendLLDP()      (every 5 s)
              │            Error      → "SWITCH DISCONNECTED", goroutine exits
              │
              └── go handleConnection(conn)     ...
```

- Messages from a given switch are handled **in order, by a single goroutine**. Per-switch state (such as the DPID) lives in a `switchConn` value owned by that goroutine, so it needs no locking.
- The `SwitchRegistry` and the `LinkRegistry` are shared by all switches and are protected by a `sync.RWMutex`.

### Link discovery (LLDP)

The controller learns how the switches are connected using LLDP (the same technique as OFDP, used by Ryu and ONOS):

1. When a switch connects, the controller installs a top-priority flow `eth_type=0x88cc → CONTROLLER`.
2. For every port, the controller sends a `PACKET_OUT` containing an LLDP frame that says "I left switch *X*, port *Y*". This happens right after the port list arrives and then every 5 seconds.
3. The switch on the other end of the cable receives the frame and sends it back to the controller in a `PACKET_IN`, together with the port it came in on.
4. Sender (from the frame) + receiver (from the `PACKET_IN`) = one link, stored in the `LinkRegistry`.

Each physical cable shows up as two unidirectional links (one per direction). Ports connected to hosts produce no links, because hosts do not send LLDP back.

LLDP frames are built and parsed by our own `internal/lldp` package. The one in libOpenflow (`protocol/lldp.go`) is broken: it writes the wrong TLVs and has no End TLV.

## Project structure

```
.
├── controller/                     # SDN controller (Go)
│   ├── cmd/controller/main.go      # Entry point
│   └── internal/
│       ├── controller/
│       │   ├── controller.go       # TCP server, connection loop, dispatch, handlers
│       │   ├── discovery.go        # LLDP flow, periodic LLDP sending, link learning
│       │   └── errors.go           # Readable logging of OpenFlow ERROR messages
│       ├── lldp/                   # Builds and parses LLDP frames (+ unit tests)
│       └── topology/
│           ├── registry.go         # Switch and port registry
│           └── topology.go         # Link registry
└── topology/
    └── topology.py                 # Mininet topology
```

## Current status

Implemented:
- [x] OpenFlow 1.5 handshake (`HELLO`, `FEATURES_REQUEST` / `FEATURES_REPLY`)
- [x] Port discovery (`MULTIPART_REQUEST` of type `PORT_DESC`)
- [x] In-memory registry of switches and ports, safe for concurrent use
- [x] Keepalive (`ECHO_REQUEST` / `ECHO_REPLY`)
- [x] Readable logging of `ERROR` messages (error type, code and the request that failed)
- [x] Table-miss flow: packets that match no flow are sent to the controller (`PACKET_IN`)
- [x] Link discovery between switches (LLDP)
- [x] One goroutine per switch; runs clean under the Go race detector

Next steps:
- [ ] Expire links that stop being seen (cable removed, switch down)
- [ ] Handle `PORT_STATUS` and remove a switch from the registry when it disconnects
- [ ] Host discovery (learn which host is behind which port)
- [ ] Forwarding: install flows along loop-free paths
- [ ] Security features (firewall/ACLs, attack detection, ...)

## Installation (Fedora)

### Base tools
```bash
sudo dnf install -y \
    git \
    python3 \
    iproute \
    iputils \
    net-tools
```

`net-tools` is required because Mininet uses `ifconfig` internally.

Optional, for inspecting OpenFlow traffic:
```bash
sudo dnf install -y tcpdump wireshark
```

### Open vSwitch
```bash
sudo dnf install -y openvswitch
sudo systemctl enable --now openvswitch
```

### Mininet
```bash
sudo dnf install -y mininet
```
> Use the Fedora package rather than `pip install mininet`. The pip package does not ship the `mnexec` binary, which Mininet needs to run.

### Go
```bash
sudo dnf install -y golang
```

### Go dependencies
Dependencies (including `antrea.io/libOpenflow`) are listed in `go.mod`. To download them:
```bash
cd controller
go mod download
```

## Running

You need two terminals, both starting from the project root.

**Terminal 1: controller**
```bash
cd controller && go run ./cmd/controller
```

**Terminal 2: Mininet topology**
```bash
sudo python3 topology/topology.py
```

### Verifying

At the Mininet prompt:
```
mininet> sh ovs-vsctl show
```
Each bridge (`s1`, `s2`, `s3`) should show `is_connected: true`.

For each switch, the controller should log `HELLO received ... version=6` (OpenFlow 1.5), then `SWITCH CONNECTED` followed by its list of ports.

A few moments later it should log one `LINK DISCOVERED` line per link direction. In this topology that is 6 lines:
```
LINK DISCOVERED: 0000000000000001:2 -> 0000000000000002:2
LINK DISCOVERED: 0000000000000002:2 -> 0000000000000001:2
LINK DISCOVERED: 0000000000000001:3 -> 0000000000000003:3
LINK DISCOVERED: 0000000000000003:3 -> 0000000000000001:3
LINK DISCOVERED: 0000000000000002:3 -> 0000000000000003:2
LINK DISCOVERED: 0000000000000003:2 -> 0000000000000002:3
```
You can compare them with the cabling reported by `mininet> net`.

To run the unit tests:
```bash
cd controller && go test ./...
```

To check for data races during development, run the controller with the race detector:
```bash
cd controller && go run -race ./cmd/controller
```

> `pingall` currently fails, and that is expected: the controller only installs the table-miss and LLDP flows, not forwarding flows yet.

### Cleaning up

If Mininet exits unexpectedly:
```bash
sudo mn -c
```
