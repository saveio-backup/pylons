package transfer

import (
	"container/list"

	"github.com/saveio/pylons/common"
	"sync"
)

// Topology represents a network topology
type Topology struct {
	nodes map[common.Address]int64
	edges map[common.Address]map[common.Address]int64
}

// Edge represents a directed edge in a graph
type Edge struct {
	NodeA    common.Address
	NodeB    common.Address
	Distance int64
}

type ShortPathTree [][]common.Address

// NewTopology creates a new topology
func NewTopology(nodes *sync.Map, edges *sync.Map) *Topology {
	t := &Topology{
		nodes: make(map[common.Address]int64),
		edges: make(map[common.Address]map[common.Address]int64),
	}

	nodes.Range(func(key, value interface{}) bool {
		t.nodes[key.(common.Address)] = -1
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
	var sp ShortPathTree
	lst := list.New()
	for n := range self.nodes {
		if n != node {
			lst.PushBack(n)
		}
	}

	var lastLstLen = lst.Len()
	for e := lst.Front(); lst.Len() != 0; e = e.Next() {
		if e == nil {
			e = lst.Front()
		}

		n := e.Value.(common.Address)
		//fmt.Println("Node | n : ", node, "| ", n)
		if _, exist1 := self.edges[node][n]; exist1 {
			var path []common.Address
			path = append(path, n)
			sp = append(sp, path)
			//fmt.Println("Remove: ", e.Value.(common.Address))
			//fmt.Println("Path: ", path)
			lst.Remove(e)
		} else {
			existPath := false
			pathCount := len(sp)
			for i := 0; i < pathCount; i++ {
				tmpPathLen := len(sp[i])
				pathLastNode := sp[i][tmpPathLen-1]
				//fmt.Println("PathLastNode |  n : ", pathLastNode, "| ", n)
				if _, exist2 := self.edges[pathLastNode]; exist2 {
					//fmt.Println("exist2")
					mp := self.edges[pathLastNode]
					//fmt.Println("mp: ", mp)
					if _, exist3 := mp[n]; exist3 {
						existPath = true
						//fmt.Println("exist3")
						path := make([]common.Address, len(sp[i]))
						copy(path, sp[i])
						path = append(path, n)
						sp = append(sp, path)
						//fmt.Println("Path: ", path)
					}
				}
			}
			if existPath {
				//fmt.Println("Remove: ", e.Value.(common.Address))
				lst.Remove(e)
			}
		}

		if e == lst.Back() && lst.Len() == lastLstLen {
			//fmt.Println(lastLstLen)
			break
		}

		lastLstLen = lst.Len()
	}
	//fmt.Println(sp)
	return sp
}
