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
	route.NewTopology(nodes, edges, nil)
	spt := route.GetShortPathTree(fromAddress, toAddress)
	for index := 0; index < len(spt); index++ {
		for _, v := range spt[index] {
			node, _ := nodes.Load(v)
			fmt.Printf("%d ", node)
		}
		fmt.Println()
	}
}

func getResultD(names []string, edgeNames [][]string, fromIndex, toIndex int) ShortPathTree {
	nodes := new(sync.Map)
	for k, n := range names {
		addr, _ := common.FromBase58(n)
		nodes.Store(addr, int64(k))
	}

	edges := new(sync.Map)
	for _, en := range edgeNames {
		NodeA := en[0]
		NodeB := en[1]
		addrA, _ := common.FromBase58(NodeA)
		addrB, _ := common.FromBase58(NodeB)

		var nodeANodeB common.EdgeId
		copy(nodeANodeB[:constants.AddrLen], addrA[:])
		copy(nodeANodeB[constants.AddrLen:], addrB[:])
		edges.Store(nodeANodeB, int64(1))
	}

	fromAddress, err := common.FromBase58(names[fromIndex])
	toAddress, err := common.FromBase58(names[toIndex])
	if err != nil {
		fmt.Println(err)
		return nil
	}

	route := &Dijkstra{}
	route.NewTopology(nodes, edges, *new([]common.Address))
	spt := route.GetShortPathTree(fromAddress, toAddress)

	from, _ := nodes.Load(fromAddress)
	to, _ := nodes.Load(toAddress)
	fmt.Printf("--- [dijkstra] path from [%d] to [%d]:\n", from, to)
	for index := 0; index < len(spt); index++ {
		for _, v := range spt[index] {
			node, _ := nodes.Load(v)
			fmt.Printf("%d ", node)
		}
		fmt.Println()
	}
	return spt
}

func TestSPTD(t *testing.T) {
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
		"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf",
		"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng",
		"AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ",
	}
	edgeNames := make([][]string, 0)
	edgeNames = append(edgeNames, []string{names[0], names[1]})
	edgeNames = append(edgeNames, []string{names[1], names[0]})
	edgeNames = append(edgeNames, []string{names[0], names[2]})
	edgeNames = append(edgeNames, []string{names[2], names[0]})
	edgeNames = append(edgeNames, []string{names[2], names[3]})
	edgeNames = append(edgeNames, []string{names[3], names[2]})
	edgeNames = append(edgeNames, []string{names[4], names[2]})
	edgeNames = append(edgeNames, []string{names[2], names[4]})
	spt1 := getResult(names, edgeNames, 1, 4)
	spt2 := getResultD(names, edgeNames, 1, 4)
	if !sliceEqual(spt1[0], spt2[0]) {
		t.Error()
	}
}

func TestSptWithSubnetD(t *testing.T) {
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
		"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf",
		"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng",
		"AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ",
		"Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC",
		"AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg",
	}
	edgeNames := make([][]string, 0)
	edgeNames = append(edgeNames, []string{names[0], names[1]})
	edgeNames = append(edgeNames, []string{names[1], names[0]})
	edgeNames = append(edgeNames, []string{names[0], names[2]})
	edgeNames = append(edgeNames, []string{names[2], names[0]})
	edgeNames = append(edgeNames, []string{names[2], names[3]})
	edgeNames = append(edgeNames, []string{names[3], names[2]})
	edgeNames = append(edgeNames, []string{names[1], names[3]})
	edgeNames = append(edgeNames, []string{names[3], names[1]})
	edgeNames = append(edgeNames, []string{names[4], names[5]})
	edgeNames = append(edgeNames, []string{names[5], names[4]})
	edgeNames = append(edgeNames, []string{names[4], names[6]})
	edgeNames = append(edgeNames, []string{names[6], names[4]})
	edgeNames = append(edgeNames, []string{names[5], names[6]})
	edgeNames = append(edgeNames, []string{names[6], names[5]})
	getResult(names, edgeNames, 1, 3)
	getResultD(names, edgeNames, 1, 3)
	spt1 := getResult(names, edgeNames, 4, 6)
	spt2 := getResultD(names, edgeNames, 4, 6)
	if !sliceEqual(spt1[0], spt2[0]) {
		t.Error()
	}
}

func TestSptWith2hopsD(t *testing.T) {
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
		"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf",
		"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng",
		"AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ",
		"Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC",
		"AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg",
	}
	edgeNames := make([][]string, 0)
	edgeNames = append(edgeNames, []string{names[0], names[1]})
	edgeNames = append(edgeNames, []string{names[1], names[0]})
	edgeNames = append(edgeNames, []string{names[0], names[2]})
	edgeNames = append(edgeNames, []string{names[2], names[0]})
	edgeNames = append(edgeNames, []string{names[2], names[3]})
	edgeNames = append(edgeNames, []string{names[3], names[2]})
	edgeNames = append(edgeNames, []string{names[1], names[3]})
	edgeNames = append(edgeNames, []string{names[3], names[1]})
	edgeNames = append(edgeNames, []string{names[4], names[0]})
	edgeNames = append(edgeNames, []string{names[0], names[4]})
	edgeNames = append(edgeNames, []string{names[2], names[5]})
	edgeNames = append(edgeNames, []string{names[5], names[2]})
	edgeNames = append(edgeNames, []string{names[3], names[6]})
	edgeNames = append(edgeNames, []string{names[6], names[3]})
	spt1 := getResult(names, edgeNames, 4, 5)
	spt2 := getResultD(names, edgeNames, 4, 5)
	if !sliceEqual(spt1[0], spt2[0]) {
		t.Error()
	}
}

func TestSptWithCircleD(t *testing.T) {
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
		"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf",
		"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng",
		"AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ",
		"Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC",
		"AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg",
	}
	edgeNames := make([][]string, 0)
	edgeNames = append(edgeNames, []string{names[0], names[1]})
	edgeNames = append(edgeNames, []string{names[1], names[0]})
	edgeNames = append(edgeNames, []string{names[1], names[2]})
	edgeNames = append(edgeNames, []string{names[2], names[1]})
	edgeNames = append(edgeNames, []string{names[2], names[3]})
	edgeNames = append(edgeNames, []string{names[3], names[2]})
	edgeNames = append(edgeNames, []string{names[3], names[4]})
	edgeNames = append(edgeNames, []string{names[4], names[3]})
	edgeNames = append(edgeNames, []string{names[4], names[5]})
	edgeNames = append(edgeNames, []string{names[5], names[4]})
	edgeNames = append(edgeNames, []string{names[5], names[6]})
	edgeNames = append(edgeNames, []string{names[6], names[5]})
	edgeNames = append(edgeNames, []string{names[6], names[0]})
	edgeNames = append(edgeNames, []string{names[0], names[6]})
	spt1 := getResult(names, edgeNames, 0, 6)
	spt2 := getResultD(names, edgeNames, 0, 6)
	if !sliceEqual(spt1[0], spt2[0]) {
		t.Error()
	}
}

func TestSptWithDiamondD(t *testing.T) {
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
		"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf",
		"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng",
		"AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ",
	}
	edgeNames := make([][]string, 0)
	edgeNames = append(edgeNames, []string{names[0], names[1]})
	edgeNames = append(edgeNames, []string{names[1], names[4]})
	edgeNames = append(edgeNames, []string{names[2], names[4]})
	edgeNames = append(edgeNames, []string{names[3], names[4]})
	edgeNames = append(edgeNames, []string{names[1], names[2]})
	edgeNames = append(edgeNames, []string{names[2], names[3]})
	edgeNames = append(edgeNames, []string{names[1], names[3]})
	spt1 := getResult(names, edgeNames, 3, 1)
	spt2 := getResultD(names, edgeNames, 3, 1)
	if !sliceEqual(spt1[0], spt2[0]) {
		t.Error()
	}
}

func TestSptWithMultiPathD(t *testing.T) {
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
		"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf",
		"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng",
		"AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ",
		"Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC",
		"AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg",
	}
	edgeNames := make([][]string, 0)
	edgeNames = append(edgeNames, []string{names[0], names[1]})
	edgeNames = append(edgeNames, []string{names[1], names[2]})
	edgeNames = append(edgeNames, []string{names[2], names[3]})
	edgeNames = append(edgeNames, []string{names[3], names[4]})
	edgeNames = append(edgeNames, []string{names[4], names[5]})
	edgeNames = append(edgeNames, []string{names[0], names[2]})
	edgeNames = append(edgeNames, []string{names[0], names[3]})
	edgeNames = append(edgeNames, []string{names[0], names[4]})
	spt1 := getResult(names, edgeNames, 5, 0)
	spt2 := getResultD(names, edgeNames, 5, 0)
	if !sliceEqual(spt1[0], spt2[0]) {
		t.Error()
	}
}

func TestBlackListWithMultiPathD(t *testing.T) {
	nodes := new(sync.Map)
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
		"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf",
		"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng",
	}
	for k, n := range names {
		addr, _ := common.FromBase58(n)
		nodes.Store(addr, int64(k))
	}

	edgeNames := make([][]string, 0)
	edgeNames = append(edgeNames, []string{names[0], names[1]})
	edgeNames = append(edgeNames, []string{names[0], names[2]})
	edgeNames = append(edgeNames, []string{names[1], names[3]})
	edgeNames = append(edgeNames, []string{names[2], names[3]})
	edgeNames = append(edgeNames, []string{names[2], names[1]})
	edgeNames = append(edgeNames, []string{names[1], names[2]})

	edgeDistance := make(map[int]int64)
	edgeDistance[0] = 1
	edgeDistance[1] = 2
	edgeDistance[2] = 1
	edgeDistance[3] = 2
	edgeDistance[4] = 1
	edgeDistance[5] = 1
	if len(edgeDistance) != len(edgeNames) {
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

	fromAddress, err := common.FromBase58(names[3])
	toAddress, err := common.FromBase58(names[0])
	if err != nil {
		fmt.Println(err)
		return
	}

	blackAddr, err := common.FromBase58(names[1])
	fmt.Println(common.ToBase58(blackAddr))
	blackList := []common.Address {blackAddr}
	//blackList := []common.Address {}

	route := &DFS{}
	route.NewTopology(nodes, edges, blackList)
	spt1 := route.GetShortPathTree(fromAddress, toAddress)

	from, _ := nodes.Load(fromAddress)
	to, _ := nodes.Load(toAddress)
	fmt.Printf("--- [dfs] path from [%d] to [%d]:\n", from, to)
	for index := 0; index < len(spt1); index++ {
		for _, v := range spt1[index] {
			node, _ := nodes.Load(v)
			fmt.Printf("%d ", node)
		}
		fmt.Println()
	}

	route2 := &Dijkstra{}
	route2.NewTopology(nodes, edges, blackList)
	spt2 := route2.GetShortPathTree(fromAddress, toAddress)

	fmt.Printf("--- [dijkstra] path from [%d] to [%d]:\n", from, to)
	for index := 0; index < len(spt2); index++ {
		for _, v := range spt2[index] {
			node, _ := nodes.Load(v)
			fmt.Printf("%d ", node)
		}
		fmt.Println()
	}

	if !sliceEqual(spt1[0], spt2[0]) {
		t.Error()
	}
}

func TestBlackListWithPairPathD(t *testing.T) {
	nodes := new(sync.Map)
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
	}
	for k, n := range names {
		addr, _ := common.FromBase58(n)
		nodes.Store(addr, int64(k))
	}

	edgeNames := make([][]string, 0)
	edgeNames = append(edgeNames, []string{names[0], names[1]})
	edgeNames = append(edgeNames, []string{names[1], names[0]})

	edgeDistance := make(map[int]int64)
	edgeDistance[0] = 1
	edgeDistance[1] = 1
	if len(edgeDistance) != len(edgeNames) {
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

	fromAddress, err := common.FromBase58(names[0])
	toAddress, err := common.FromBase58(names[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	blackAddr, err := common.FromBase58(names[1])
	fmt.Println(common.ToBase58(blackAddr))
	blackList := []common.Address {blackAddr}
	//blackList := []common.Address {}

	route := &DFS{}
	route.NewTopology(nodes, edges, blackList)
	spt1 := route.GetShortPathTree(fromAddress, toAddress)

	from, _ := nodes.Load(fromAddress)
	to, _ := nodes.Load(toAddress)
	fmt.Printf("--- [dfs] path from [%d] to [%d]:\n", from, to)
	for index := 0; index < len(spt1); index++ {
		for _, v := range spt1[index] {
			node, _ := nodes.Load(v)
			fmt.Printf("%d ", node)
		}
		fmt.Println()
	}

	route2 := &Dijkstra{}
	route2.NewTopology(nodes, edges, blackList)
	spt2 := route2.GetShortPathTree(fromAddress, toAddress)

	fmt.Printf("--- [dijkstra] path from [%d] to [%d]:\n", from, to)
	for index := 0; index < len(spt2); index++ {
		for _, v := range spt2[index] {
			node, _ := nodes.Load(v)
			fmt.Printf("%d ", node)
		}
		fmt.Println()
	}

	if !sliceEqual(spt1[0], spt2[0]) {
		t.Error()
	}
}