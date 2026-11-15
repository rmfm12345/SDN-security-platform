#!/usr/bin/env python3

from mininet.topo import Topo
from mininet.net import Mininet
from mininet.cli import CLI
from mininet.log import setLogLevel
from mininet.node import OVSSwitch


class SecurityTopology(Topo):
    def build(self):
        h1 = self.addHost("h1")
        h2 = self.addHost("h2")
        h3 = self.addHost("h3")

        s1 = self.addSwitch("s1")
        s2 = self.addSwitch("s2")
        s3 = self.addSwitch("s3")

        self.addLink(h1, s1)
        self.addLink(h2, s2)
        self.addLink(h3, s3)

        self.addLink(s1, s2)
        self.addLink(s2, s3)
        self.addLink(s1, s3)


def configure_switch(switch):
    switch.cmd(
        "ovs-vsctl set bridge %s protocols=OpenFlow15"
        % switch.name
    )

    switch.cmd(
        "ovs-vsctl set-fail-mode %s secure"
        % switch.name
    )

    switch.cmd(
        "ovs-vsctl set-controller %s tcp:127.0.0.1:6653"
        % switch.name
    )


def main():
    topology = SecurityTopology()

    net = Mininet(
        topo=topology,
        switch=OVSSwitch,
        controller=None,
        autoSetMacs=True,
    )

    net.start()

    for switch in net.switches:
        configure_switch(switch)

    CLI(net)
    net.stop()


if __name__ == "__main__":
    setLogLevel("info")
    main()