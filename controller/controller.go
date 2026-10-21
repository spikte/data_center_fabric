package main

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v3"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

func run(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Topology
type DCTopology struct {
	Name        string
	Spines      []Router
	Leafs       []Router
	Servers     []Server
	Spines2Id   map[string]int
	Leafs2Id    map[string]int
	Servers2Id  map[string]int
	Topology    Graph
	Links       []NetworkLink
	Underlay    string
	Overlay     string
	SameSession bool
	ClabPath    string
}

func NewDCTopology() DCTopology {
	return DCTopology{
		Topology:   NewGraph(),
		Spines2Id:  make(map[string]int),
		Leafs2Id:   make(map[string]int),
		Servers2Id: make(map[string]int),
	}
}
func (topology *DCTopology) GetSpine(name string) *Router {
	return &topology.Spines[topology.Spines2Id[name]]
}
func (topology *DCTopology) GetLeaf(name string) *Router {
	return &topology.Leafs[topology.Leafs2Id[name]]
}
func (topology *DCTopology) GetServer(name string) *Server {
	return &topology.Servers[topology.Servers2Id[name]]
}
func (topology *DCTopology) AddSpine(spine Router) {
	topology.Spines = append(topology.Spines, spine)
	topology.Spines2Id[spine.NetEl.Name] = len(topology.Spines) - 1
}
func (topology *DCTopology) AddLeaf(leaf Router) {
	topology.Leafs = append(topology.Leafs, leaf)
	topology.Leafs2Id[leaf.NetEl.Name] = len(topology.Leafs) - 1
}
func (topology *DCTopology) AddServer(server Server) {
	topology.Servers = append(topology.Servers, server)
	topology.Servers2Id[server.NetEl.Name] = len(topology.Servers) - 1
}
func (topology *DCTopology) GetNetEl(name string) *NetworkElement {
	if strings.HasPrefix(name, "spine") {
		return &topology.GetSpine(name).NetEl
	} else if strings.HasPrefix(name, "leaf") {
		return &topology.GetLeaf(name).NetEl
	} else if strings.HasPrefix(name, "server") {
		return &topology.GetServer(name).NetEl
	}
	return nil
}
func (topology *DCTopology) FillData(config YMLConfig) {
	globalId := 1
	for i := 0; i < config.Spines; i++ {
		name := fmt.Sprintf("spine%d", i+1)
		topology.AddSpine(Router{
			NetEl: NetworkElement{
				Name:           name,
				GlobalId:       globalId,
				Id:             i + 1,
				ManagementAddr: netip.MustParsePrefix(fmt.Sprintf("172.16.0.%d/32", i+1)),
			},
			Type:     Spine,
			AS:       65000,
			RouterId: netip.MustParsePrefix(fmt.Sprintf("10.117.0.%d/32", globalId)),
			Loopback: netip.MustParsePrefix(fmt.Sprintf("10.117.0.%d/32", globalId)),
		})
		globalId++
		topology.Topology.AddNode(name)
	}
	for i := 0; i < config.Servers; i++ {
		name := fmt.Sprintf("leaf%d", i+1)
		topology.AddLeaf(Router{
			NetEl: NetworkElement{
				Name:           name,
				GlobalId:       globalId,
				Id:             i + 1,
				ManagementAddr: netip.MustParsePrefix(fmt.Sprintf("172.16.1.%d/32", i+1)),
			},
			Type:     Leaf,
			AS:       65000 + globalId,
			RouterId: netip.MustParsePrefix(fmt.Sprintf("10.117.0.%d/32", globalId)),
			Loopback: netip.MustParsePrefix(fmt.Sprintf("10.117.0.%d/32", globalId)),
		})
		globalId++
		topology.Topology.AddNode(name)
	}
	for i := 0; i < config.Servers; i++ {
		name := fmt.Sprintf("server%d", i+1)
		topology.AddServer(Server{
			NetEl: NetworkElement{
				Name:           name,
				GlobalId:       globalId,
				Id:             i + 1,
				ManagementAddr: netip.MustParsePrefix(fmt.Sprintf("172.16.2.%d/32", i+1)),
			},
		})
		globalId++
		topology.Topology.AddNode(name)
	}
}
func (topology *DCTopology) FillLink() {
	for i := range topology.Leafs {
		leaf := &topology.Leafs[i]
		for j := range topology.Spines {
			spine := &topology.Spines[j]
			// Add the edge to the graph topology
			topology.Topology.AddEdgeLabel(leaf.NetEl.Name, spine.NetEl.Name)
			// Interface name
			intSpineId := len(spine.NetEl.Interfaces) + 1
			intLeafId := len(leaf.NetEl.Interfaces) + 1
			intSpine := fmt.Sprintf("eth%d", intSpineId)
			intLeaf := fmt.Sprintf("eth%d", intLeafId)
			// Interface P2P address
			ipSpine := netip.MustParsePrefix(fmt.Sprintf("10.%d.%d.0/31", spine.NetEl.Id, leaf.NetEl.Id))
			ipLeaf := netip.MustParsePrefix(fmt.Sprintf("10.%d.%d.1/31", spine.NetEl.Id, leaf.NetEl.Id))
			// Add interface to the nodes
			spine.NetEl.Interfaces = append(spine.NetEl.Interfaces, Interface{
				Name: intSpine,
				MTU:  9500,
				IP:   ipSpine,
			})
			leaf.NetEl.Interfaces = append(leaf.NetEl.Interfaces, Interface{
				Name: intLeaf,
				MTU:  9500,
				IP:   ipLeaf,
			})
			// Create the link and add it to the topology and nodes
			link := NetworkLink{
				HostA:        spine.NetEl.Name,
				InterfaceA:   intSpine,
				InterfaceAId: intSpineId - 1,
				HostB:        leaf.NetEl.Name,
				InterfaceB:   intLeaf,
				InterfaceBId: intLeafId - 1,
			}
			topology.Links = append(topology.Links, link)
			leaf.NetEl.Neighbors = append(leaf.NetEl.Neighbors, &link)
			spine.NetEl.Neighbors = append(spine.NetEl.Neighbors, &link)
		}
	}
	for name, _ := range topology.Servers {
		server := &topology.Servers[name]
		leaf := &topology.Leafs[name]

		topology.Topology.AddEdgeLabel(leaf.NetEl.Name, server.NetEl.Name)
		// Interface name
		intLeafId := len(leaf.NetEl.Interfaces) + 1
		intServerId := len(server.NetEl.Interfaces) + 1
		intLeaf := fmt.Sprintf("eth%d", intLeafId)
		intServer := fmt.Sprintf("eth%d", intServerId)
		// Interface P2P address
		ipLeaf := netip.MustParsePrefix(fmt.Sprintf("10.%d.%d.1/31", 10+leaf.NetEl.Id, server.NetEl.Id))
		ipServer := netip.MustParsePrefix(fmt.Sprintf("10.%d.%d.0/31", 10+leaf.NetEl.Id, server.NetEl.Id))
		// Add interface to the nodes
		server.NetEl.Interfaces = append(server.NetEl.Interfaces, Interface{
			Name: intServer,
			MTU:  1500,
			IP:   ipServer,
		})
		leaf.NetEl.Interfaces = append(leaf.NetEl.Interfaces, Interface{
			Name: intLeaf,
			MTU:  1500,
			IP:   ipLeaf,
		})
		// Create the link and add it to the topology and nodes
		link := NetworkLink{
			HostA:        server.NetEl.Name,
			InterfaceA:   intServer,
			InterfaceAId: intServerId - 1,
			HostB:        leaf.NetEl.Name,
			InterfaceB:   intLeaf,
			InterfaceBId: intLeafId - 1,
		}
		topology.Links = append(topology.Links, link)
		leaf.NetEl.Neighbors = append(leaf.NetEl.Neighbors, &link)
		server.NetEl.Neighbors = append(server.NetEl.Neighbors, &link)
	}
}
func (topology *DCTopology) FillVlan(config YMLConfig) {
	for tenantId, tenant := range config.Tenants {
		tenantVNIId := 1_000_000 + tenantId*100_000
		for i, segment := range tenant.Segments {
			gateway := netip.AddrFrom4([4]byte{10, byte(200 + tenantId), byte(i), 254})
			tenantIP := netip.AddrFrom4([4]byte{10, byte(200 + tenantId), byte(i), 0})
			vni := tenantVNIId + segment.VLAN
			for _, serverName := range segment.Hosts {
				// Server
				server := topology.GetServer(serverName)
				server.VMs = append(server.VMs, VM{
					TenantName:  tenant.Name,
					TenantId:    tenantId + 1,
					TenantVNIId: tenantVNIId,
					VLAN:        segment.VLAN,
					VNI:         vni,
					IP:          tenantIP.Next(),
					Gateway:     gateway,
					Prefix:      24,
				})
				tenantIP = tenantIP.Next()
				// Leaf
				for _, link := range server.NetEl.Neighbors {
					var leaf *Router
					var intName string
					if link.HostA == serverName {
						leaf = topology.GetLeaf(link.HostB)
						intName = link.InterfaceB
					} else {
						leaf = topology.GetLeaf(link.HostA)
						intName = link.InterfaceA
					}
					leaf.VLAN = append(leaf.VLAN, map[string]any{
						"vlan":          segment.VLAN,
						"vni":           vni,
						"interface":     intName,
						"l3vni-ip":      gateway.String() + "/24",
						"l3vni-mac":     net.HardwareAddr{0x02, 0x00, 0x00, 0x00, byte(segment.VLAN), 0xFE}.String(),
						"tenant-name":   tenant.Name,
						"tenant-vni-id": tenantVNIId,
						"tenant-id":     tenantId + 1,
					})
				}
			}
		}
	}
}
func (topology *DCTopology) WriteContainerLab(path string) {
	content := map[string]any{
		"name": topology.Name,
		"mgmt": map[string]any{
			"network":     topology.Name + "-mgmt",
			"ipv4-subnet": "172.16.0.0/22",
			"ipv4-gw":     "172.16.3.254",
		},
		"topology": map[string]any{
			"nodes": map[string]any{},
			"links": []map[string]any{},
		},
	}

	nodes := content["topology"].(map[string]any)["nodes"].(map[string]any)
	for _, spine := range topology.Spines {
		nodes[spine.NetEl.Name] = map[string]string{
			"kind":      "linux",
			"image":     "frr-ssh",
			"mgmt-ipv4": spine.NetEl.ManagementAddr.Addr().String(),
		}
	}
	for _, leaf := range topology.Leafs {
		nodes[leaf.NetEl.Name] = map[string]string{
			"kind":      "linux",
			"image":     "frr-ssh",
			"mgmt-ipv4": leaf.NetEl.ManagementAddr.Addr().String(),
		}
	}
	for _, server := range topology.Servers {
		hostDir := fmt.Sprintf("/var/lib/clab-datacenter/%s", server.NetEl.Name)
		contDir := "/var/lib/docker"
		nodes[server.NetEl.Name] = map[string]any{
			"kind":      "linux",
			"image":     "ubuntu-ssh",
			"mgmt-ipv4": server.NetEl.ManagementAddr.Addr().String(),
			"binds":     []string{hostDir + ":" + contDir},
		}
		if err := os.MkdirAll(hostDir, 0755); err != nil {
			panic(err)
		}
	}
	var links []map[string]any
	for _, link := range topology.Links {
		links = append(links, map[string]any{
			"endpoints": []string{
				link.HostA + ":" + link.InterfaceA,
				link.HostB + ":" + link.InterfaceB,
			},
		})
	}
	content["topology"].(map[string]any)["links"] = links

	writeYAML(content, path)
}
func (topology *DCTopology) WriteAnsibleInventoryHosts(path string) {
	content := map[string]any{
		"all": map[string]any{
			"vars": map[string]any{
				"ansible_user":      "root",
				"ansible_password":  "root",
				"underlay":          topology.Underlay,
				"overlay":           topology.Overlay,
				"ebgp_same_session": topology.SameSession,
			},
			"children": map[string]any{
				"spines":  map[string]any{"hosts": map[string]any{}},
				"leafs":   map[string]any{"hosts": map[string]any{}},
				"servers": map[string]any{"hosts": map[string]any{}},
			},
		},
	}
	spines := content["all"].(map[string]any)["children"].(map[string]any)["spines"].(map[string]any)["hosts"].(map[string]any)
	for _, spine := range topology.Spines {
		spines[spine.NetEl.Name] = map[string]string{
			"ansible_host": spine.NetEl.ManagementAddr.Addr().String(),
		}
	}
	leafs := content["all"].(map[string]any)["children"].(map[string]any)["leafs"].(map[string]any)["hosts"].(map[string]any)
	for _, leaf := range topology.Leafs {
		leafs[leaf.NetEl.Name] = map[string]string{
			"ansible_host": leaf.NetEl.ManagementAddr.Addr().String(),
		}
	}
	servers := content["all"].(map[string]any)["children"].(map[string]any)["servers"].(map[string]any)["hosts"].(map[string]any)
	for _, server := range topology.Servers {
		servers[server.NetEl.Name] = map[string]string{
			"ansible_host": server.NetEl.ManagementAddr.Addr().String(),
		}
	}

	writeYAML(content, path)
}
func (topology *DCTopology) WriteAnsibleInventoryP2P(path string) {
	var p2p_links []map[string]any
	for _, link := range topology.Links {
		netElA := topology.GetNetEl(link.HostA)
		netElB := topology.GetNetEl(link.HostB)
		intA := netElA.Interfaces[link.InterfaceAId]
		intB := netElB.Interfaces[link.InterfaceBId]
		p2p_links = append(p2p_links, map[string]any{
			"name": link.HostA + "-" + link.HostB,
			"mtu":  min(intA.MTU, intB.MTU),
			"endpoints": []map[string]string{
				{
					"host":      link.HostA,
					"interface": link.InterfaceA,
					"ip":        intA.IP.String(),
				},
				{
					"host":      link.HostB,
					"interface": link.InterfaceB,
					"ip":        intB.IP.String(),
				},
			},
		})
	}
	content := map[string]any{
		"p2p_links": p2p_links,
	}
	writeYAML(content, path)
}
func (topology *DCTopology) GetInterfacesYAML(netEl *NetworkElement) map[string]any {
	interfacesYAML := make(map[string]any)
	for _, int := range netEl.Interfaces {
		interfacesYAML[int.Name] = map[string]any{
			"ip":     int.IP.Addr().String(),
			"prefix": int.IP.Bits(),
			"mtu":    int.MTU,
		}
	}
	return interfacesYAML
}
func (topology *DCTopology) GetNeighborsYAML(netEl *NetworkElement) []map[string]any {
	var neighborsYAML []map[string]any
	for _, neighbor := range netEl.Neighbors {
		var name string
		var ip string
		if neighbor.HostA == netEl.Name {
			name = neighbor.HostB
			ip = topology.GetNetEl(name).Interfaces[neighbor.InterfaceBId].IP.Addr().String()
		} else {
			name = neighbor.HostA
			ip = topology.GetNetEl(name).Interfaces[neighbor.InterfaceAId].IP.Addr().String()
		}
		neighborsYAML = append(neighborsYAML, map[string]any{
			"name": name,
			"ip":   ip,
		})
	}
	return neighborsYAML
}
func (topology *DCTopology) GetVMsYAML(vms *[]VM) []map[string]any {
	var vmsYAML []map[string]any
	for i, vm := range *vms {
		vmsYAML = append(vmsYAML, map[string]any{
			"id":          i,
			"tenant_name": vm.TenantName,
			"tenant_id":   vm.TenantId,
			"vlan":        vm.VLAN,
			"vni":         vm.VNI,
			"ip":          vm.IP.String() + fmt.Sprintf("/%d", vm.Prefix),
			"gateway":     vm.Gateway,
		})
	}
	return vmsYAML
}
func (topology *DCTopology) WriteAnsibleInventoryHostVars(path string) {
	for _, spine := range topology.Spines {
		content := map[string]any{
			"global_id":  spine.NetEl.GlobalId,
			"id":         spine.NetEl.Id,
			"type":       "spine",
			"as-number":  spine.AS,
			"router-id":  spine.RouterId.Addr().String(),
			"loopback":   spine.Loopback.Addr().String(),
			"interfaces": topology.GetInterfacesYAML(&spine.NetEl),
			"neighbors":  topology.GetNeighborsYAML(&spine.NetEl),
		}
		writeYAML(content, filepath.Join(path, spine.NetEl.Name+".yml"))
	}
	for _, leaf := range topology.Leafs {
		content := map[string]any{
			"global_id":  leaf.NetEl.GlobalId,
			"id":         leaf.NetEl.Id,
			"type":       "leaf",
			"as-number":  leaf.AS,
			"router-id":  leaf.RouterId.Addr().String(),
			"loopback":   leaf.Loopback.Addr().String(),
			"interfaces": topology.GetInterfacesYAML(&leaf.NetEl),
			"neighbors":  topology.GetNeighborsYAML(&leaf.NetEl),
			"vlan_vni":   leaf.VLAN,
		}
		writeYAML(content, filepath.Join(path, leaf.NetEl.Name+".yml"))
	}
	for _, server := range topology.Servers {
		content := map[string]any{
			"global_id":  server.NetEl.GlobalId,
			"id":         server.NetEl.Id,
			"type":       "server",
			"interfaces": topology.GetInterfacesYAML(&server.NetEl),
			"neighbors":  topology.GetNeighborsYAML(&server.NetEl),
			"vm":         topology.GetVMsYAML(&server.VMs),
		}
		writeYAML(content, filepath.Join(path, server.NetEl.Name+".yml"))
	}
}

func (topology *DCTopology) Build() error {
	topology.WriteContainerLab("./datacenter.clab.yml")
	topology.WriteAnsibleInventoryHosts("./ansible/inventory/hosts.yml")
	topology.WriteAnsibleInventoryP2P("./ansible/inventory/group_vars/all.yml")
	topology.WriteAnsibleInventoryHostVars("./ansible/inventory/host_vars/")
	collections := []string{
		"ansible.posix",
		"community.general",
		"community.docker",
	}
	for _, collection := range collections {
		if err := run("./ansible", "ansible-galaxy", "collection", "install", collection); err != nil {
			return fmt.Errorf("Couldn't install ansible collection %s: %w", collection, err)
		}

	}
	return nil
}
func (topology *DCTopology) Start() error {
	if err := run("./", "containerlab", "deploy", "-t", topology.ClabPath); err != nil {
		return fmt.Errorf("Couldn't deploy containerlab: %w", err)
	}

	if err := os.Chdir("ansible"); err != nil {
		topology.Stop()
		return fmt.Errorf("Couldn't cd to the ansible directory: %w", err)
	}

	playbooks := []string{
		"playbook/setup_p2p.yml",
		"playbook/setup_network_devices.yml",
	}
	for _, playbook := range playbooks {
		if err := run("./", "ansible-playbook", playbook); err != nil {
			return fmt.Errorf("Couldn't run playbook %s: %w", playbook, err)
		}
	}
	return nil
}
func (topology *DCTopology) Stop() {
	// Destroy the containerlab
	if err := run("./", "containerlab", "destroy", "-t", topology.ClabPath, "--cleanup"); err != nil {
		fmt.Printf("Error while destroying containerlab: %v", err)
	}
	// Delete known .ssh
	for _, spine := range topology.Spines {
		err := spine.NetEl.RemoveSSHHost()
		if err != nil {
			fmt.Printf("Error while removing ssh host: %v", err)
			continue
		}
	}
	for _, leaf := range topology.Leafs {
		err := leaf.NetEl.RemoveSSHHost()
		if err != nil {
			fmt.Printf("Error while removing ssh host: %v", err)
			continue
		}
	}
	for _, server := range topology.Servers {
		err := server.NetEl.RemoveSSHHost()
		if err != nil {
			fmt.Printf("Error while removing ssh host: %v", err)
			continue
		}
	}
}

func main() {
	path := flag.String("path", "", "Configuration file path")
	action := flag.String("action", "", "Action to do")
	allowedAction := []string{"Build", "Start", "Stop", "Restart"}

	flag.Parse()

	if *path == "" {
		fmt.Fprintln(os.Stderr, "Error: -path is required")
		flag.Usage()
		os.Exit(1)
	}
	if !slices.Contains(allowedAction, *action) {
		fmt.Fprintf(os.Stderr, "Invalid action %s. Allowed: %v\n", *action, allowedAction)
		os.Exit(1)
	}

	data, err := os.ReadFile(*path)
	if err != nil {
		panic(err)
	}
	var config YMLConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		panic(err)
	}
	topology := NewDCTopology()
	topology.Name = config.Name
	topology.Underlay = config.Underlay
	topology.Overlay = config.Overlay
	topology.SameSession = config.SameSession
	topology.ClabPath = strings.TrimSuffix(*path, ".yml") + ".clab.yml"
	topology.FillData(config)
	topology.FillLink()
	topology.FillVlan(config)

	switch *action {
	case "Build":
		topology.Build()
	case "Start":
		topology.Start()
	case "Stop":
		topology.Stop()
	case "Restart":
		topology.Stop()
		topology.Start()
	}
}
