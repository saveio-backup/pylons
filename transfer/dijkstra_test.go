package transfer

import (
	"testing"

	"fmt"
	chainComm "github.com/oniio/oniChain/common"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
)

func TestSPT(t *testing.T) {
	nodes := make(map[common.Address]int64)
	names := []string{
		"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq",
		"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS",
		"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf",
		"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng",
		"AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ",
	}
	for _, n := range names {
		addr, _ := chainComm.AddressFromBase58(n)
		nodes[common.Address(addr)] = 0
	}
	fmt.Println(nodes)

	edges := make(map[common.EdgeId]int64)

	edgeNames := make([][]string, 0)
	edgeNames = append(edgeNames, []string{"AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq", "AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS"})
	edgeNames = append(edgeNames, []string{"AYMnqA65pJFKAbbpD8hi5gdNDBmeFBy5hS", "AGeTrARjozPVLhuzMxZq36THMtvsrZNAHq"})
	edgeNames = append(edgeNames, []string{"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf", "AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng"})
	edgeNames = append(edgeNames, []string{"AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng", "AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf"})
	edgeNames = append(edgeNames, []string{"AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ", "AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf"})
	edgeNames = append(edgeNames, []string{"AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf", "AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ"})

	for _, en := range edgeNames {
		NodeA := en[0]
		NodeB := en[1]
		addrA, _ := chainComm.AddressFromBase58(NodeA)
		addrB, _ := chainComm.AddressFromBase58(NodeB)

		var nodeANodeB common.EdgeId
		copy(nodeANodeB[:constants.ADDR_LEN], addrA[:])
		copy(nodeANodeB[constants.ADDR_LEN:], addrB[:])
		edges[nodeANodeB] = 1

		var nodeBNodeA common.EdgeId
		copy(nodeBNodeA[:constants.ADDR_LEN], addrB[:])
		copy(nodeBNodeA[constants.ADDR_LEN:], addrA[:])
		edges[nodeBNodeA] = 1
	}
	fmt.Println(edges)

	toAddress, err := chainComm.AddressFromBase58("AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ")
	if err != nil {
		t.Fatal(err)
	}

	top := NewTopology(nodes, edges)
	fmt.Println("TOP: ", top)
	spt := top.GetShortPath(common.Address(toAddress))
	fmt.Println("SPT:", spt)

}
