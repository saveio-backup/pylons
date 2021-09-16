package route

import (
	"github.com/saveio/pylons/common"
	"math"
	"sync"
)

type Dijkstra struct {
	DirectedGraph *DirectedGraph
}

func (dijkstra *Dijkstra) NewTopology(nodes, edges *sync.Map, opts ...interface{}) {
	directedGraph := NewDirectGraph(nodes, edges)
	dijkstra.DirectedGraph = directedGraph
}

func (dijkstra *Dijkstra) GetShortPathTree(from, to common.Address, opts ...interface{}) ShortPathTree {
	return dijkstra.DirectedGraph.GetPairShortestPath(from,to)
}

type DirectedGraph map[common.Address]Vertex

type Vertex struct {
	From    map[common.Address]int64
	To      map[common.Address]int64
}

func NewDirectGraph(nodes *sync.Map, edges *sync.Map) *DirectedGraph {
	dg := make(DirectedGraph)
	addrs := make([]common.Address, 0)
	topology := NewTopology(nodes, edges, addrs)
	for node := range topology.nodes {
		vernice := Vertex{
			From:    make(map[common.Address]int64),
			To:      make(map[common.Address]int64),
		}
		dg[node] = vernice
	}
	edges.Range(func(key, value interface{}) bool {
		addr1 := key.(common.EdgeId).GetAddr1()
		addr2 := key.(common.EdgeId).GetAddr2()
		dg[addr1].From[addr2] = value.(int64)
		dg[addr2].To[addr1] = value.(int64)
		return true
	})
	return &dg
}

func (g *DirectedGraph) GetPairShortestPath(from, to common.Address) ShortPathTree {
	path := make([]common.Address, 0)
	dist := map[common.Address]int64 {from: 0}
	prev := map[common.Address]int64 {from: 0}
	g.searchPathDijkstra(dist, prev, from, to, &path)
	// reverse path for interface format
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return ShortPathTree{path}
}

func (g *DirectedGraph) searchPathDijkstra(dist, prev map[common.Address]int64, from, to common.Address, path *[]common.Address) {
	// save path
	*path = append(*path, from)
	// already find shortest path
	if from == to {
		return
	}
	// find all reachable node
	for node, distance := range (*g)[from].To {
		if _, ok := dist[node]; !ok {
			dist[node] = math.MaxInt64
		}
		if _,ok := prev[node]; !ok {
			dist[node] = func(a, b int64) int64 {
				if a < b {
					return a
				}
				return b
			}(dist[from] + distance, dist[node])
		}
	}
	// find nearest node
	var nearestNode common.Address
	minDistance := int64(math.MaxInt64)
	for node, distance := range dist {
		if _, ok := prev[node]; !ok &&  distance < minDistance {
			nearestNode = node
			minDistance = distance
		}
	}
	// if no nearest node from this 'from'
	if minDistance == int64(math.MaxInt64) {
		return
	}
	// update prev set
	if _, ok := prev[nearestNode]; !ok {
		prev[nearestNode] = int64(len(prev))
	}
	// remove node from path if not in target
	if _, ok := (*g)[from].To[nearestNode]; !ok {
		*path = (*path)[:len(*path)-1]
	}
	g.searchPathDijkstra(dist, prev, nearestNode, to, path)
}
