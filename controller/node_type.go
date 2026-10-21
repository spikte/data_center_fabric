package main

import (
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
)

// Network types
type Interface struct {
	Name string
	MTU  int
	IP   netip.Prefix
}
type NetworkLink struct {
	HostA        string
	InterfaceA   string
	InterfaceAId int
	HostB        string
	InterfaceB   string
	InterfaceBId int
}
type NetworkElement struct {
	Name           string
	GlobalId       int
	Id             int
	Interfaces     []Interface
	Neighbors      []*NetworkLink
	ManagementAddr netip.Prefix
}

func (netEl *NetworkElement) RemoveSSHHost() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("Couldn't resolve home directory: %w", err)
	}
	knownHosts := filepath.Join(home, ".ssh", "known_hosts")
	ip := netEl.ManagementAddr.Addr().String()
	if ip == "" {
		return fmt.Errorf("No management ip for node %s", netEl.Name)
	}
	fmt.Printf("Removing SSH key for %s (%s)\n", netEl.Name, ip)
	if err := run(".", "ssh-keygen", "-f", knownHosts, "-R", ip); err != nil {
		return fmt.Errorf("ssh-keygen failed for node %s: %w", netEl.Name, err)
	}
	return nil
}

type FabricRouterType int

const (
	Spine FabricRouterType = iota
	Leaf
)

type Router struct {
	NetEl    NetworkElement
	Type     FabricRouterType
	AS       int
	RouterId netip.Prefix
	Loopback netip.Prefix
	VLAN     []map[string]any
}
type VM struct {
	TenantName  string
	TenantId    int
	TenantVNIId int
	VLAN        int
	VNI         int
	IP          netip.Addr
	Gateway     netip.Addr
	Prefix      int
}
type Server struct {
	NetEl NetworkElement
	VMs   []VM
}
