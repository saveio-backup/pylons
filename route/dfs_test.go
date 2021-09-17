package route

import (
	"fmt"
	"sync"
	"testing"

	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
)

func getResult(names []string, edgeNames [][]string, fromIndex, toIndex int) ShortPathTree {
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

	route := &DFS{}
	route.NewTopology(nodes, edges, *new([]common.Address))
	spt := route.GetShortPathTree(fromAddress, toAddress)

	from, _ := nodes.Load(fromAddress)
	to, _ := nodes.Load(toAddress)
	fmt.Printf("--- [dfs] path from [%d] to [%d]:\n", from, to)
	for index := 0; index < len(spt); index++ {
		for _, v := range spt[index] {
			node, _ := nodes.Load(v)
			fmt.Printf("%d ", node)
		}
		fmt.Println()
	}
	return spt
}

func TestSPT(t *testing.T) {
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
	getResult(names, edgeNames, 1, 4)
}

func TestSptWithSubnet(t *testing.T) {
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
	getResult(names, edgeNames, 4, 6)
}
func TestSptWith2hops(t *testing.T) {
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
	getResult(names, edgeNames, 4, 5)
}

func TestSptWithCircle(t *testing.T) {
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
	getResult(names, edgeNames, 0, 6)
}

func TestSptWithDiamond(t *testing.T) {
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
	getResult(names, edgeNames, 3, 1)
}

func TestSptWithMultiPath(t *testing.T) {
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
	getResult(names, edgeNames, 5, 0)
}

func TestPrevAddr(t *testing.T) {
	nodes := new(sync.Map)
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
		"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf",
		"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng",
		"AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ",
		"Ac54scP31i6h5zUsYGPegLf2yUSCK74KYC",
		"AQAz1RTZLW6ptervbNzs29rXKvKJuFNxMg",
	}
	for k, n := range names {
		addr, _ := common.FromBase58(n)
		nodes.Store(addr, int64(k))
	}

	edges := new(sync.Map)
	edgeNames := make([][]string, 0)
	edgeNames = append(edgeNames, []string{names[0], names[1]})
	edgeNames = append(edgeNames, []string{names[1], names[2]})
	edgeNames = append(edgeNames, []string{names[2], names[3]})
	edgeNames = append(edgeNames, []string{names[3], names[4]})
	edgeNames = append(edgeNames, []string{names[4], names[5]})
	edgeNames = append(edgeNames, []string{names[0], names[2]})
	edgeNames = append(edgeNames, []string{names[0], names[3]})
	edgeNames = append(edgeNames, []string{names[0], names[4]})

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

	fromAddress, err := common.FromBase58(names[5])
	toAddress, err := common.FromBase58(names[0])
	if err != nil {
		fmt.Println(err)
		return
	}

	route := &DFS{}
	route.NewTopology(nodes, edges, nil)
	spt := route.GetShortPathTree(fromAddress, toAddress)

	from, _ := nodes.Load(fromAddress)
	to, _ := nodes.Load(toAddress)
	fmt.Printf("--- path from [%d] to [%d]:\n", from, to)
	for index := 0; index < len(spt); index++ {
		for _, v := range spt[index] {
			node, _ := nodes.Load(v)
			fmt.Printf("%d ", node)
		}
		fmt.Println()
	}
}

