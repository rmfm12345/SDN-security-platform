#!/usr/bin/env python3

from mininet.topo import Topo
from mininet.net import Mininet
from mininet.cli import CLI
from mininet.log import setLogLevel


class SecurityTopology(Topo):

    def build(self):
        # Hosts
        h1 = self.addHost("h1")
        h2 = self.addHost("h2")
        h3 = self.addHost("h3")

        # Switches
        s1 = self.addSwitch("s1")
        s2 = self.addSwitch("s2")
        s3 = self.addSwitch("s3")

        # Host-to-switch links
        self.addLink(h1, s1)
        self.addLink(h2, s2)
        self.addLink(h3, s3)

        # Switch-to-switch links
        self.addLink(s1, s2)
        self.addLink(s2, s3)
        self.addLink(s1, s3)


def main():
    topology = SecurityTopology()

    net = Mininet(topo=topology, controller=None)

    net.start()

    CLI(net)

    net.stop()


if __name__ == "__main__":
    setLogLevel("info")
    main()