package transfer

import (
	"container/list"
	"github.com/oniio/oniChannel/common"
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
func NewTopology(nodes map[common.Address]int64, edges map[common.EdgeId]int64) *Topology {
	t := &Topology{
		nodes: make(map[common.Address]int64),
		edges: make(map[common.Address]map[common.Address]int64),
	}

	for n := range nodes {
		t.nodes[n] = -1
	}

	for e, d := range edges {
		addr1 := e.GetAddr1()
		addr2 := e.GetAddr2()
		if _, ok := t.edges[addr1]; !ok {
			t.edges[addr1] = make(map[common.Address]int64)
		}

		t.edges[addr1][addr2] = d
	}

	return t
}

func (self *Topology) GetShortPath(node common.Address) ShortPathTree {
	var sp ShortPathTree
	lst := list.New()
	for n := range self.nodes  {
		if n != node {
			lst.PushBack(n)
		}
	}

	//for e := lst.Front(); e != nil ; e = e.Next()  {
		//fmt.Println(e.Value.(common.Address))
	//}
	//fmt.Println()

	var lastLstLen = lst.Len()
	for e := lst.Front(); lst.Len() != 0; e = e.Next(){
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
			exist3 := false
			pathCount := len(sp)
			for i := 0; i < pathCount; i++  {
				tmpPathLen := len(sp[i])
				pathLastNode := sp[i][tmpPathLen - 1]
				//fmt.Println("PathLastNode |  n : ", pathLastNode, "| ", n)
				if _, exist2 := self.edges[pathLastNode]; exist2 {
					//fmt.Println("exist2")
					mp := self.edges[pathLastNode]
					//fmt.Println("mp: ", mp)
					if _, exist3 = mp[n]; exist3 {
						//fmt.Println("exist3")
						path := make([]common.Address, len(sp[i]))
						copy(path, sp[i])
						path = append(path, n)
						sp = append(sp, path)
						//fmt.Println("Path: ", path)
					}
				}
			}
			if exist3 {
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
