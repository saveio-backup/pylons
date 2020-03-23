package transfer

import (
	"fmt"
	"sync"
	"testing"

	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
)

func TestSPT(t *testing.T) {
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
		"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf",
		"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng",
		"AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ",
	}
	alias := make(map[common.Address]string)
	nodes := new(sync.Map)
	edges := new(sync.Map)
	for k, n := range names {
		addr, _ := common.FromBase58(n)
		nodes.Store(addr, int64(1))
		name := fmt.Sprintf("%s%d", "node", k)
		alias[addr] = name
		//fmt.Println("alias[addr] = ", alias[addr])
	}
	for k, en := range alias {
		fmt.Printf("%s --- %s \n", common.ToBase58(k), en)
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

	for _, en := range edgeNames {
		NodeA := en[0]
		NodeB := en[1]
		addrA, _ := common.FromBase58(NodeA)
		addrB, _ := common.FromBase58(NodeB)

		var nodeANodeB common.EdgeId
		copy(nodeANodeB[:constants.AddrLen], addrA[:])
		copy(nodeANodeB[constants.AddrLen:], addrB[:])
		edges.Store(nodeANodeB, int64(1))
		fmt.Printf("add edge:%s-%s\n", alias[addrA], alias[addrB])
	}

	toAddress, err := common.FromBase58("AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ")
	if err != nil {
		t.Fatal(err)
	}
	top := NewTopology(nodes, edges, common.Address{})
	//fmt.Println("TOP: ", top)
	spt := top.GetShortPath(common.Address(toAddress))
	fmt.Printf("path to %s:\n", alias[toAddress])
	for index := 0; index < len(spt); index++ {
		for _, v := range spt[index] {
			fmt.Printf("%s ", alias[v])
		}
		fmt.Println()
	}
}

func TestDijWithSubnet(t *testing.T) {
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
		"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf",
		"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng",
		"AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ",
		"Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC",
		"AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg",
	}
	alias := make(map[common.Address]string)
	nodes := new(sync.Map)
	edges := new(sync.Map)
	for k, n := range names {
		addr, _ := common.FromBase58(n)
		nodes.Store(addr, int64(1))
		name := fmt.Sprintf("%s%d", "node", k)
		alias[addr] = name
		//fmt.Println("alias[addr] = ", alias[addr])
	}
	for k, en := range alias {
		fmt.Printf("%s --- %s \n", common.ToBase58(k), en)
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

	for _, en := range edgeNames {
		NodeA := en[0]
		NodeB := en[1]
		addrA, _ := common.FromBase58(NodeA)
		addrB, _ := common.FromBase58(NodeB)

		var nodeANodeB common.EdgeId
		copy(nodeANodeB[:constants.AddrLen], addrA[:])
		copy(nodeANodeB[constants.AddrLen:], addrB[:])
		edges.Store(nodeANodeB, int64(1))
		fmt.Printf("add edge:%s-%s\n", alias[addrA], alias[addrB])
	}

	toAddress, err := common.FromBase58("AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ")
	if err != nil {
		t.Fatal(err)
	}
	top := NewTopology(nodes, edges, common.Address{})
	spt := top.GetShortPath(common.Address(toAddress))
	fmt.Printf("path to %s:\n", alias[toAddress])
	for index := 0; index < len(spt); index++ {
		for _, v := range spt[index] {
			fmt.Printf("%s ", alias[v])
		}
		fmt.Println()
	}
}
func TestDijWith2hops(t *testing.T) {
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
		"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf",
		"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng",
		"AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ",
		"Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC",
		"AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg",
	}
	alias := make(map[common.Address]string)
	nodes := new(sync.Map)
	edges := new(sync.Map)
	for k, n := range names {
		addr, _ := common.FromBase58(n)
		nodes.Store(addr, int64(1))
		name := fmt.Sprintf("%s%d", "node", k)
		alias[addr] = name
		//fmt.Println("alias[addr] = ", alias[addr])
	}
	for k, en := range alias {
		fmt.Printf("%s --- %s \n", common.ToBase58(k), en)
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

	for _, en := range edgeNames {
		NodeA := en[0]
		NodeB := en[1]
		addrA, _ := common.FromBase58(NodeA)
		addrB, _ := common.FromBase58(NodeB)

		var nodeANodeB common.EdgeId
		copy(nodeANodeB[:constants.AddrLen], addrA[:])
		copy(nodeANodeB[constants.AddrLen:], addrB[:])
		edges.Store(nodeANodeB, int64(1))
		fmt.Printf("add edge:%s-%s\n", alias[addrA], alias[addrB])
	}

	toAddress, err := common.FromBase58("AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ")
	if err != nil {
		t.Fatal(err)
	}
	top := NewTopology(nodes, edges, common.Address{})
	spt := top.GetShortPath(common.Address(toAddress))
	fmt.Printf("path to %s:\n", alias[toAddress])
	for index := 0; index < len(spt); index++ {
		for _, v := range spt[index] {
			fmt.Printf("%s ", alias[v])
		}
		fmt.Println()
	}
}

func TestDijWithCircle(t *testing.T) {
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
		"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf",
		"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng",
		"AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ",
		"Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC",
		"AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg",
	}
	alias := make(map[common.Address]string)
	nodes := new(sync.Map)
	edges := new(sync.Map)
	for k, n := range names {
		addr, _ := common.FromBase58(n)
		nodes.Store(addr, int64(1))
		name := fmt.Sprintf("%s%d", "node", k)
		alias[addr] = name
		//fmt.Println("alias[addr] = ", alias[addr])
	}
	for k, en := range alias {
		fmt.Printf("%s --- %s \n", common.ToBase58(k), en)
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

	for _, en := range edgeNames {
		NodeA := en[0]
		NodeB := en[1]
		addrA, _ := common.FromBase58(NodeA)
		addrB, _ := common.FromBase58(NodeB)

		var nodeANodeB common.EdgeId
		copy(nodeANodeB[:constants.AddrLen], addrA[:])
		copy(nodeANodeB[constants.AddrLen:], addrB[:])
		edges.Store(nodeANodeB, int64(1))
		fmt.Printf("add edge:%s-%s\n", alias[addrA], alias[addrB])
	}

	toAddress, err := common.FromBase58("AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ")
	if err != nil {
		t.Fatal(err)
	}
	top := NewTopology(nodes, edges, common.Address{})
	spt := top.GetShortPath(common.Address(toAddress))
	fmt.Printf("path to %s:\n", alias[toAddress])
	for index := 0; index < len(spt); index++ {
		for _, v := range spt[index] {
			fmt.Printf("%s ", alias[v])
		}
		fmt.Println()
	}
}

func TestDijWithDiamond(t *testing.T) {
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
		"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf",
		"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng",
		"AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ",
	}
	alias := make(map[common.Address]string)
	nodes := new(sync.Map)
	edges := new(sync.Map)
	for k, n := range names {
		addr, _ := common.FromBase58(n)
		nodes.Store(addr, int64(1))
		name := fmt.Sprintf("%s%d", "node", k)
		alias[addr] = name
		//fmt.Println("alias[addr] = ", alias[addr])
	}
	for k, en := range alias {
		fmt.Printf("%s --- %s \n", common.ToBase58(k), en)
	}
	edgeNames := make([][]string, 0)
	edgeNames = append(edgeNames, []string{names[0], names[1]})

	edgeNames = append(edgeNames, []string{names[1], names[4]})
	edgeNames = append(edgeNames, []string{names[2], names[4]})
	edgeNames = append(edgeNames, []string{names[3], names[4]})

	edgeNames = append(edgeNames, []string{names[1], names[2]})
	edgeNames = append(edgeNames, []string{names[2], names[3]})
	edgeNames = append(edgeNames, []string{names[1], names[3]})

	for _, en := range edgeNames {
		NodeA := en[0]
		NodeB := en[1]
		addrA, _ := common.FromBase58(NodeA)
		addrB, _ := common.FromBase58(NodeB)

		var nodeANodeB common.EdgeId
		copy(nodeANodeB[:constants.AddrLen], addrA[:])
		copy(nodeANodeB[constants.AddrLen:], addrB[:])
		edges.Store(nodeANodeB, int64(1))
		fmt.Printf("add edge:%s-%s\n", alias[addrA], alias[addrB])
	}

	fromAddress, err := common.FromBase58("AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq")
	if err != nil {
		t.Fatal(err)
	}
	top := NewTopology(nodes, edges, common.Address{})
	spt := top.GetShortPath(common.Address(fromAddress))
	fmt.Printf("path to %s:\n", alias[fromAddress])
	for index := 0; index < len(spt); index++ {
		for _, v := range spt[index] {
			fmt.Printf("%s ", alias[v])
		}
		fmt.Println()
	}
}
