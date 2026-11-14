# SDN Network Security

Controlador SDN escrito de raiz em **Go**, com foco em segurança de rede, testado numa rede emulada com **Mininet** e **Open vSwitch** sobre **OpenFlow 1.3**.

## Arquitetura

**Plano de controlo:** cada switch liga-se ao controlador em `tcp:127.0.0.1:6653` (OpenFlow 1.3).

**Plano de dados:**
```
           ┌────────────────────────┐
           │                        │
  h1 ──── s1 ──────── s2 ──────── s3 ──── h3
                      │
                      h2
```

- 3 hosts (`h1`, `h2`, `h3`), cada um ligado a um switch.
- 3 switches OVS ligados em triângulo (`s1–s2`, `s2–s3`, `s1–s3`), o que cria um ciclo na topologia.
- Os switches estão em `fail-mode secure`, ou seja, só encaminham tráfego segundo os flows instalados pelo controlador.

## Estrutura do projeto

```
.
├── controller/                     # Controlador SDN (Go)
│   ├── cmd/controller/main.go      # Ponto de entrada
│   └── internal/
│       ├── controller/             # Handlers OpenFlow
│       └── topology/               # Registo de switches e portas
└── topology/
    └── topology.py                 # Topologia Mininet
```

## Estado atual

Implementado:
- [x] Handshake OpenFlow (`HELLO`, `FEATURES_REQUEST` / `FEATURES_REPLY`)
- [x] Descoberta de portas (`MULTIPART_REQUEST` do tipo `PORT_DESC`)
- [x] Registo em memória de switches e portas
- [x] Keepalive (`ECHO_REQUEST` / `ECHO_REPLY`)

Próximos passos:
- [ ] Proteger os mapas partilhados com `sync.Mutex` (concorrência)
- [ ] Limpar o estado quando um switch se desliga
- [ ] Handler para mensagens `ERROR`
- [ ] Tratar `PACKET_IN` e instalar flows (encaminhamento)
- [ ] Descoberta de links entre switches (LLDP)
- [ ] Funcionalidades de segurança (firewall/ACLs, deteção de ataques, ...)

## Instalação (Fedora)

### Ferramentas base
```bash
sudo dnf install -y \
    git \
    gcc \
    gcc-c++ \
    make \
    autoconf \
    automake \
    libtool \
    pkg-config \
    python3 \
    python3-pip \
    iproute \
    iproute-tc \
    iputils \
    tcpdump \
    wireshark \
    iperf3 \
    net-tools
```

### Open vSwitch
```bash
sudo dnf install -y openvswitch
sudo systemctl enable --now openvswitch
```

### Mininet
```bash
pip install mininet
```
> Nota: a instalação via `pip` pode causar problemas no Fedora.

### Go
```bash
sudo dnf install -y golang
```

### Dependências Go
As dependências (incluindo `github.com/netrack/openflow`) estão no `go.mod`. Para as descarregar:
```bash
cd controller
go mod download
```

## Execução

São precisos dois terminais, ambos a partir da raiz do projeto.

**Terminal 1: controlador**
```bash
cd controller && go run ./cmd/controller
```

**Terminal 2: topologia Mininet**
```bash
sudo python3 topology/topology.py
```

### Verificar

No prompt do Mininet:
```
mininet> sh ovs-vsctl show
```
Cada bridge (`s1`, `s2`, `s3`) deve mostrar `is_connected: true`.

O controlador deve registar, para cada switch, `SWITCH CONNECTED` e a lista de portas.

> Neste momento o `pingall` falha, e isso é esperado: o controlador ainda não instala flows.

### Limpar o ambiente

Se o Mininet terminar de forma inesperada:
```bash
sudo mn -c
```
