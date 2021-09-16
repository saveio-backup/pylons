package route

import (
	"sort"
	"sync"

	"github.com/saveio/pylons/common"
)

type DFS struct {
	Topology *Topology
}

func (dfs *DFS) NewTopology(nodes, edges *sync.Map, opts ...interface{}) {
	prevAddrs := make([]common.Address, 0)
	if opts != nil && len(opts) > 0 {
		if opts[0] != nil {
			switch opts[0].(type) {
			case []common.Address:
				prevAddrs = opts[0].([]common.Address)
			}
		}
	}
	topology := NewTopology(nodes, edges, prevAddrs)
	dfs.Topology = topology
}

func (dns *DFS) GetShortPathTree(from, to common.Address, opts ...interface{}) ShortPathTree {
	return dns.Topology.GetPairPathSorted(from, to)
}

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

// NewTopology creates a new topology
func NewTopology(nodes *sync.Map, edges *sync.Map, previousAddrs []common.Address) *Topology {
	t := &Topology{
		nodes: make(map[common.Address]int64),
		edges: make(map[common.Address]map[common.Address]int64),
	}

	nodes.Range(func(key, value interface{}) bool {
		tmpAddr := key.(common.Address)
		if !common.AddressContains(previousAddrs, tmpAddr) {
			t.nodes[tmpAddr] = value.(int64)
		}
		return true
	})

	edges.Range(func(key, value interface{}) bool {
		addr1 := key.(common.EdgeId).GetAddr1()
		addr2 := key.(common.EdgeId).GetAddr2()
		if !common.AddressContains(previousAddrs, addr1) && !common.AddressContains(previousAddrs, addr2) {
			if _, ok := t.edges[addr1]; !ok {
				t.edges[addr1] = make(map[common.Address]int64)
			}
			t.edges[addr1][addr2] = value.(int64)
		}
		return true
	})
	return t
}

func (self *Topology) GetAllPath(from common.Address) ShortPathTree {
	if 0 == len(self.nodes) || 0 == len(self.edges) {
		return [][]common.Address{}
	}
	var path []common.Address
	self.searchPathDFS(from, path)
	return self.sp
}

func (self *Topology) GetAllPathSorted(from common.Address) ShortPathTree {
	self.GetAllPath(from)
	sort.Sort(self.sp)
	return self.sp
}

func (self *Topology) searchPathDFS(from common.Address, path []common.Address) {
	path = append(path, from)
	// path may repeat when recurse return to upper layer
	for _, v := range self.sp {
		if sliceEqual(v, path) {
			return
		}
	}
	pathTemp := make([]common.Address, len(path))
	copy(pathTemp, path)
	self.sp = append(self.sp, pathTemp)
	for n := range self.edges[from] {
		// skip nodes already in path record
		walked := false
		for _, v := range path {
			if n == v {
				walked = true
				break
			}
		}
		if walked {
			continue
		}
		self.searchPathDFS(n, path)
	}
	return
}

func (self *Topology) GetPairPath(from, to common.Address) ShortPathTree {
	if 0 == len(self.nodes) || 0 == len(self.edges) {
		return [][]common.Address{}
	}
	var path []common.Address
	self.searchPathDFSTarget(from, to, path)
	return self.sp
}

func (self *Topology) GetPairPathSorted(from, to common.Address) ShortPathTree {
	self.GetPairPath(from, to)
	sort.Sort(self.sp)
	return self.sp
}

func (self *Topology) searchPathDFSTarget(from, to common.Address, path []common.Address) {
	path = append(path, to)
	if to == from {
		pathTemp := make([]common.Address, len(path))
		copy(pathTemp, path)
		self.sp = append(self.sp, pathTemp)
		return
	}
	for _, v := range self.sp {
		if sliceEqual(v, path) {
			return
		}
	}
	for n := range self.edges[to] {
		walked := false
		for _, v := range path {
			if n == v {
				walked = true
				break
			}
		}
		if walked {
			continue
		}
		self.searchPathDFSTarget(from, n, path)
	}
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
