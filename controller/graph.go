package main

// Graph: Actually not used but I keep it I might need it for some viz
type Graph struct {
	Adj      [][]int
	Label    []string
	Label2Id map[string]int
}

func NewGraph() Graph {
	return Graph{
		Adj:      [][]int{},
		Label:    []string{},
		Label2Id: make(map[string]int),
	}
}
func (g *Graph) AddNode(label string) {
	if len(g.Adj) == 0 {
		g.Adj = append(g.Adj, []int{0})
	} else {
		for i := 0; i < len(g.Adj); i++ {
			g.Adj[i] = append(g.Adj[i], 0)
		}
		newRow := make([]int, len(g.Adj[0]))
		g.Adj = append(g.Adj, newRow)
	}
	g.Label = append(g.Label, label)
	g.Label2Id[label] = len(g.Label) - 1
}
func (g *Graph) AddEdge(a int, b int) {
	g.Adj[a][b] = 1
	g.Adj[b][a] = 1
}
func (g *Graph) AddEdgeLabel(a string, b string) {
	g.AddEdge(g.Label2Id[a], g.Label2Id[b])
}
func (g *Graph) RmEdge(a int, b int) {
	g.Adj[a][b] = 0
	g.Adj[b][a] = 0
}
func (g *Graph) RmEdgeStr(a string, b string) {
	g.RmEdge(g.Label2Id[a], g.Label2Id[b])
}
func (g *Graph) Neighbors(n int) []int {
	var neighbors []int
	for i := 0; i < len(g.Adj); i++ {
		if g.Adj[n][i] != 0 {
			neighbors = append(neighbors, i)
		}
	}
	return neighbors
}
func (g *Graph) NeighborsLabel(label string) []string {
	neighbors := g.Neighbors(g.Label2Id[label])
	var neighborsLabel []string
	for _, n := range neighbors {
		neighborsLabel = append(neighborsLabel, g.Label[n])
	}
	return neighborsLabel
}
