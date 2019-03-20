package transfer

import (
	"container/list"
)

// Topology represents a network topology
type Topology struct {
	nodes map[string]int64
	edges map[string]map[string]int64
}

// Edge represents a directed edge in a graph
type Edge struct {
	NodeA    string
	NodeB    string
	Distance int64
}

type ShortPathTree [][]string

// NewTopology creates a new topology
func NewTopology(nodes []string, edges []Edge) *Topology {
	t := &Topology{
		nodes: make(map[string]int64),
		edges: make(map[string]map[string]int64),
	}

	for _, n := range nodes {
		t.nodes[n] = -1
	}

	for _, e := range edges {
		if _, ok := t.edges[e.NodeA]; !ok {
			t.edges[e.NodeA] = make(map[string]int64)
		}

		t.edges[e.NodeA][e.NodeB] = e.Distance
	}

	return t
}

func (self *Topology) GetShortPath(node string) ShortPathTree {
	var sp ShortPathTree
	lst := list.New()
	for n := range self.nodes  {
		if n != node {
			lst.PushBack(n)
		}
	}

	//for e := lst.Front(); e != nil ; e = e.Next()  {
	//	fmt.Println(e.Value.(string))
	//}
	//fmt.Println()

	var lastLstLen = lst.Len()
	for e := lst.Front(); lst.Len() != 0; e = e.Next(){
		if e == nil {
			e = lst.Front()
		}

		n := e.Value.(string)
		//fmt.Println("Node | n : ", node, "| ", n)
		if _, exist1 := self.edges[node][n]; exist1 {
			var path []string
			path = append(path, n)
			sp = append(sp, path)
			//fmt.Println("Remove: ", e.Value.(string))
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
						path := make([]string, len(sp[i]))
						copy(path, sp[i])
						path = append(path, n)
						sp = append(sp, path)
						//fmt.Println("Path: ", path)
					}
				}
			}
			if exist3 {
				//fmt.Println("Remove: ", e.Value.(string))
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
