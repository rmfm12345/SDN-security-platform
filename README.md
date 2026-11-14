# SDN Network Security

An SDN controller written from scratch in **Go**, focused on network security, and tested on an emulated network built with **Mininet** and **Open vSwitch** over **OpenFlow 1.3**.

## Architecture

**Control plane:** every switch connects to the controller at `tcp:127.0.0.1:6653` (OpenFlow 1.3).

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

## Project structure

```
.
├── controller/                     # SDN controller (Go)
│   ├── cmd/controller/main.go      # Entry point
│   └── internal/
│       ├── controller/             # OpenFlow handlers
│       └── topology/               # Switch and port registry
└── topology/
    └── topology.py                 # Mininet topology
```

## Current status

Implemented:
- [x] OpenFlow handshake (`HELLO`, `FEATURES_REQUEST` / `FEATURES_REPLY`)
- [x] Port discovery (`MULTIPART_REQUEST` of type `PORT_DESC`)
- [x] In-memory registry of switches and ports
- [x] Keepalive (`ECHO_REQUEST` / `ECHO_REPLY`)

Next steps:
- [ ] Protect shared maps with `sync.Mutex` (concurrency)
- [ ] Clean up state when a switch disconnects
- [ ] Handler for `ERROR` messages
- [ ] Handle `PACKET_IN` and install flows (forwarding)
- [ ] Link discovery between switches (LLDP)
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
Dependencies (including `github.com/netrack/openflow`) are listed in `go.mod`. To download them:
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

For each switch, the controller should log `SWITCH CONNECTED` followed by its list of ports.

> `pingall` currently fails, and that is expected: the controller does not install any flows yet.

### Cleaning up

If Mininet exits unexpectedly:
```bash
sudo mn -c
```
