#Ferramentas base
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


#instalar Open vSwitch
sudo dnf install -y openvswitch

sudo systemctl enable --now openvswitch


#instalar Mininet
pip install mininet (pode causar problemas no fedora)


#Instalar Go
sudo dnf install -y golang


$Instalar biblioteca Go
go get github.com/netrack/openflow