package route

import (
	"fmt"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
	"sync"
	"testing"
)

func TestBasic(t *testing.T) {
	// reference: https://www.cartagena99.com/recursos/alumnos/apuntes/dijkstra_algorithm.pdf
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
		"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf",
		"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng",
		"AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ",
	}
	nodes := new(sync.Map)

	for k, n := range names {
		addr, _ := common.FromBase58(n)
		nodes.Store(addr, int64(k))
	}
	edgeNames := make([][]string, 0)
	edgeNames = append(edgeNames, []string{names[1], names[0]})
	edgeNames = append(edgeNames, []string{names[2], names[0]})
	edgeNames = append(edgeNames, []string{names[2], names[1]})
	edgeNames = append(edgeNames, []string{names[3], names[1]})
	edgeNames = append(edgeNames, []string{names[1], names[2]})
	edgeNames = append(edgeNames, []string{names[3], names[2]})
	edgeNames = append(edgeNames, []string{names[4], names[2]})
	edgeNames = append(edgeNames, []string{names[4], names[3]})
	edgeNames = append(edgeNames, []string{names[3], names[4]})

	edgeDistance := make(map[int]int64)
	edgeDistance[0] = 10
	edgeDistance[1] = 3
	edgeDistance[2] = 1
	edgeDistance[3] = 2
	edgeDistance[4] = 4
	edgeDistance[5] = 8
	edgeDistance[6] = 2
	edgeDistance[7] = 7
	edgeDistance[8] = 9
	if (len(edgeDistance) != len(edgeNames)) {
		return
	}

	edges := new(sync.Map)
	for i, en := range edgeNames {
		NodeA := en[0]
		NodeB := en[1]
		addrA, _ := common.FromBase58(NodeA)
		addrB, _ := common.FromBase58(NodeB)

		var nodeANodeB common.EdgeId
		copy(nodeANodeB[:constants.AddrLen], addrA[:])
		copy(nodeANodeB[constants.AddrLen:], addrB[:])
		edges.Store(nodeANodeB, edgeDistance[i])
	}

	fromIndex := 0
	toIndex := 3
	fromAddress, err := common.FromBase58(names[fromIndex])
	toAddress, err := common.FromBase58(names[toIndex])
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("--- from [%d] to [%d] \n", fromIndex, toIndex)
	route := &Dijkstra{}
	route.NewTopology(nodes, edges)
	spt := route.GetShortPathTree(fromAddress, toAddress)
	for index := 0; index < len(spt); index++ {
		for _, v := range spt[index] {
			node, _ := nodes.Load(v)
			fmt.Printf("%d ", node)
		}
		fmt.Println()
	}
}
