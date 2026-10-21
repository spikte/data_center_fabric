package main

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type YMLSegment struct {
	VLAN  int
	Hosts []string
}
type YMLTenant struct {
	Name     string
	Segments []YMLSegment
}
type YMLConfig struct {
	Name                   string
	ManagementSubnet       string `yaml:"mgmt-subnet"`
	Servers                int
	Leafs                  int
	MaxServerPerLeaf       int `yaml:"max-server-per-leaf"`
	Spines                 int
	Underlay               string
	Overlay                string
	SameSession            bool
	RedudantServerLeafLink int `yaml:"redundant-server-uplink"`
	Tenants                []YMLTenant
}

func writeYAML(content map[string]any, path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		panic(err)
	}
	data, err := yaml.Marshal(content)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, data, 0755); err != nil {
		panic(err)
	}
}
