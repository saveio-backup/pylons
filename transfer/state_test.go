package transfer

import (
	"encoding/json"
	"fmt"
	"github.com/oniio/oniChannel/common"
	"testing"
)

type TokenNetwork struct {
	Nodes map[common.Address]int64
	Edges map[common.EdgeId]int64
}

func TestTokenNetworkGraphMarshal(t *testing.T) {
	var a TokenNetwork
	a.Nodes = make(map[common.Address]int64)
	a.Edges = make(map[common.EdgeId]int64)

	var addr common.Address
	a.Nodes[addr] = 1

	var edge common.EdgeId
	a.Edges[edge] = 2

	data, err := json.Marshal(a)
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Println(data)

	var b TokenNetwork
	b.Nodes = make(map[common.Address]int64)
	b.Edges = make(map[common.EdgeId]int64)

	err = json.Unmarshal(data, &b)
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Println(b)
}
