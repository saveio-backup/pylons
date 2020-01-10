package transfer

import (
	"sync"

	"github.com/saveio/pylons/common"
)

// Topology represents a network topology
type Topology struct {
	nodes map[common.Address]int64
	edges map[common.Address]map[common.Address]int64
	sp    ShortPathTree
}

// Edge represents a directed edge in a graph
type Edge struct {
	NodeA    common.Address
	NodeB    common.Address
	Distance int64
}

const (
	INT_MAX = 1<<32 - 1
)

type ShortPathTree [][]common.Address

// NewTopology creates a new topology
func NewTopology(nodes *sync.Map, edges *sync.Map) *Topology {
	t := &Topology{
		nodes: make(map[common.Address]int64),
		edges: make(map[common.Address]map[common.Address]int64),
	}

	nodes.Range(func(key, value interface{}) bool {
		t.nodes[key.(common.Address)] = value.(int64)
		return true
	})

	edges.Range(func(key, value interface{}) bool {
		addr1 := key.(common.EdgeId).GetAddr1()
		addr2 := key.(common.EdgeId).GetAddr2()
		if _, ok := t.edges[addr1]; !ok {
			t.edges[addr1] = make(map[common.Address]int64)
		}

		t.edges[addr1][addr2] = value.(int64)
		return true
	})
	return t
}

func (self *Topology) GetShortPath(node common.Address) ShortPathTree {
	if 0 == len(self.nodes) || 0 == len(self.edges) {
		return [][]common.Address{}
	}
	var path []common.Address
	self.searchPath(node, path)
	return self.sp
}

func (self *Topology) searchPath(node common.Address, path []common.Address) {
	path = append(path, node)
	for _, v := range self.sp {
		if sliceEqual(v, path) {
			return
		}
	}
	self.sp = append(self.sp, path)
	for n := range self.edges[node] {
		//fmt.Printf("int64 node %s loop\n", common.ToBase58(node))
		walked := false
		for _, v := range path {
			if n == v {
				//fmt.Printf("node %s exist break\n", common.ToBase58(n))
				walked = true
			}
		}
		if walked {
			continue
		}
		self.searchPath(n, path)
	}
	return
}
func sliceEqual(a, b []common.Address) bool {
	if len(a) != len(b) {
		return false
	}

	if (a == nil) != (b == nil) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}
