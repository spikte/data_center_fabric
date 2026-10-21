# Datacenter simulation with containerlab

This project aims at simulating a datacenter network using containerlab. It uses Go and Ansible to configure the lab.
It supports:
- Configurable number of servers and spines (for the moment there is one leaf per server)
- Configurable overlay/underlay:
    * eBGP/eBGP using the same session
    * eBGP/eBGP using different session
- Configurable tenants

![](./doc/configuration.png)

Docker images used:
- FRRouting for the leafs and spines
- Ubuntu for the servers
- nicolaka/netshoot for the VMs inside the servers

## Setup

You need to have go and ansible installed on you machine.
Then you need to compile the controller:

```bash
cd controller/
go build .
```

## Configuration

First configure the datacenter as you wish in a `.yml` file:

```yml
name: data-center
mgmt-subnet: 172.16.0.0/22
servers: 7
spines: 2
underlay: eBGP
overlay: eBGP
samesession: false
redundant-server-leaf-link: 0

tenants:
  - name: A
    segments:
      - vlan: 10
        hosts: ["server1", "server2", "server3", "server4"]
      - vlan: 20
        hosts: ["server1", "server2", "server3", "server4"]
  - name: B
    segments:
      - vlan: 110
        hosts: ["server1", "server2", "server3", "server4"]
      - vlan: 120
        hosts: ["server1", "server2", "server3", "server4"]
```

Then use the controller to build the project. It will create the containerlab topology file and fill the ansible inventory:
```bash
./controller/dc_controller -path datacenter.yml -action Build
```

## Running the lab

Start the lab:
```bash
./controller/dc_controller -path datacenter.yml -action Start
```

The controller also supports `Stop` and `Restart` action.
