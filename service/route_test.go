package service

import (
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/pylons/transfer"
	"sync"
	"testing"
)

func TestCalculateEdgeWeightByPenalty(t *testing.T) {
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
	edgeNames = append(edgeNames, []string{names[0], names[1]})
	edgeNames = append(edgeNames, []string{names[1], names[4]})
	edgeNames = append(edgeNames, []string{names[0], names[2]})
	edgeNames = append(edgeNames, []string{names[2], names[3]})
	edgeNames = append(edgeNames, []string{names[3], names[4]})

	edges := new(sync.Map)
	for _, en := range edgeNames {
		NodeA := en[0]
		NodeB := en[1]
		addrA, _ := common.FromBase58(NodeA)
		addrB, _ := common.FromBase58(NodeB)
		var nodeANodeB common.EdgeId
		copy(nodeANodeB[:constants.AddrLen], addrA[:])
		copy(nodeANodeB[constants.AddrLen:], addrB[:])
		edges.Store(nodeANodeB, 1)
	}

	feeMap := make(map[common.Address]*transfer.FeeScheduleState)
	b, _ := common.FromBase58(names[1])
	feeMap[b] = &transfer.FeeScheduleState{
		Flat:             2,
		Proportional:     2000000,
	}
	c, _ := common.FromBase58(names[2])
	feeMap[c] = &transfer.FeeScheduleState{
		Flat:             1,
		Proportional:     1000000,
	}
	d, _ := common.FromBase58(names[3])
	feeMap[d] = &transfer.FeeScheduleState{
		Flat:             1,
		Proportional:     1000000,
	}

	type args struct {
		edges            *sync.Map
		feePenalty       float64
		diversityPenalty float64
		amount           common.TokenAmount
		visited          int
		feeMap			 map[common.Address]*transfer.FeeScheduleState
	}
	tests := []struct {
		name string
		args args
		want *sync.Map
	}{
		{
			args: args{
				edges:            edges,
				feePenalty:       1,
				diversityPenalty: 1,
				amount:           1000,
				visited:          0,
				feeMap:           feeMap,
			},
		},
		{
			args: args{
				edges:            edges,
				feePenalty:       1,
				diversityPenalty: 1,
				amount:           1000,
				visited:          1,
				feeMap:           feeMap,
			},
		},
		{
			args: args{
				edges:            edges,
				feePenalty:       1,
				diversityPenalty: 1,
				amount:           1000,
				visited:          2,
				feeMap:           feeMap,
			},
		},
		{
			args: args{
				edges:            edges,
				feePenalty:       2,
				diversityPenalty: 1,
				amount:           1000,
				visited:          0,
				feeMap:           feeMap,
			},
		},
		{
			args: args{
				edges:            edges,
				feePenalty:       2,
				diversityPenalty: 1,
				amount:           1000,
				visited:          1,
				feeMap:           feeMap,
			},
		},
		{
			args: args{
				edges:            edges,
				feePenalty:       2,
				diversityPenalty: 1,
				amount:           1000,
				visited:          2,
				feeMap:           feeMap,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateEdgeWeightByPenalty(tt.args.edges, tt.args.amount, tt.args.visited, tt.args.feePenalty,
				tt.args.diversityPenalty, tt.args.feeMap)
			if got == nil {
				t.Fatal(nil)
			}
			got.Range(func(key, value interface{}) bool {
				id := key.(common.EdgeId)
				from, _ := nodes.Load(id.GetAddr1())
				to, _ := nodes.Load(id.GetAddr2())
				t.Log(from, to, value)
				return true
			})
		})
	}
}

func TestCalculateEdgeWeightByFee(t *testing.T) {
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
	edgeNames = append(edgeNames, []string{names[0], names[1]})
	edgeNames = append(edgeNames, []string{names[1], names[4]})
	edgeNames = append(edgeNames, []string{names[0], names[2]})
	edgeNames = append(edgeNames, []string{names[0], names[3]})
	edgeNames = append(edgeNames, []string{names[3], names[4]})

	edges := new(sync.Map)
	for _, en := range edgeNames {
		NodeA := en[0]
		NodeB := en[1]
		addrA, _ := common.FromBase58(NodeA)
		addrB, _ := common.FromBase58(NodeB)
		var nodeANodeB common.EdgeId
		copy(nodeANodeB[:constants.AddrLen], addrA[:])
		copy(nodeANodeB[constants.AddrLen:], addrB[:])
		edges.Store(nodeANodeB, 1)
	}

	feeMap := make(map[common.Address]*transfer.FeeScheduleState)
	b, _ := common.FromBase58(names[1])
	feeMap[b] = &transfer.FeeScheduleState{
		Flat:             2,
		Proportional:     2000000,
	}
	c, _ := common.FromBase58(names[2])
	feeMap[c] = &transfer.FeeScheduleState{
		Flat:             1,
		Proportional:     1000000,
	}
	d, _ := common.FromBase58(names[3])
	feeMap[d] = &transfer.FeeScheduleState{
		Flat:             1,
		Proportional:     1000000,
	}

	type args struct {
		edges            *sync.Map
		feePenalty       float64
		diversityPenalty float64
		amount           common.TokenAmount
		visited          int
		feeMap			 map[common.Address]*transfer.FeeScheduleState
	}
	tests := []struct {
		name string
		args args
		want *sync.Map
	}{
		{
			args: args{
				edges:            edges,
				amount:           1,
				feeMap:           feeMap,
			},
		},
		{
			args: args{
				edges:            edges,
				amount:           10,
				feeMap:           feeMap,
			},
		},
		{
			args: args{
				edges:            edges,
				amount:           100,
				feeMap:           feeMap,
			},
		},
		{
			args: args{
				edges:            edges,
				amount:           1000,
				feeMap:           feeMap,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateEdgeWeightByFee(tt.args.edges, tt.args.amount, tt.args.feeMap)
			if got == nil {
				t.Fatal(nil)
			}
			got.Range(func(key, value interface{}) bool {
				id := key.(common.EdgeId)
				from, _ := nodes.Load(id.GetAddr1())
				to, _ := nodes.Load(id.GetAddr2())
				t.Log(from, to, value)
				return true
			})
		})
	}
}
