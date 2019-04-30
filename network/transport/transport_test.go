package transport

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/transfer"
)

var (
	protocol  = "udp"
	node1Addr = "127.0.0.1:3001"
	node2Addr = "127.0.0.1:3002"
	node3Addr = "127.0.0.1:3003"
)

type TestChannel struct{}

func (t *TestChannel) OnMessage(message proto.Message, str string) {}
func (t *TestChannel) Sign(message interface{}) error {
	return nil
}
func (t *TestChannel) HandleStateChange(stateChange transfer.StateChange) []transfer.Event {
	return nil
}
func (t *TestChannel) Get(nodeAddress common.Address) string {
	return ""
}
func (t *TestChannel) StateFromChannel() *transfer.ChainState {
	return nil
}

var node1AccountAddress = [20]byte{1}
var node2AccountAddress = [20]byte{2}
var node3AccountAddress = [20]byte{3}

var addressToIPMap = map[[20]byte]string{
	node1AccountAddress: "127.0.0.1:3001",
	node2AccountAddress: "127.0.0.1:3002",
	node3AccountAddress: "127.0.0.1:3003",
}

func CheckConnectedNodes(peerMap *sync.Map, expected []string) error {
	for _, addr := range expected {
		_, ok := peerMap.Load(addr)
		if !ok {
			return fmt.Errorf("%s is not found in the active peer map", addr)
		}
	}

	return nil
}

func TestConnect(t *testing.T) {
	var err error
	node1 := NewTransport(protocol)
	node2 := NewTransport(protocol)
	node3 := NewTransport(protocol)

	node1.SetAddress(node1Addr)
	node2.SetAddress(node2Addr)
	node3.SetAddress(node3Addr)
	tc := &TestChannel{}
	err = node1.Start(tc)
	if err != nil {
		t.Error(err)
	}
	err = node2.Start(tc)
	if err != nil {
		t.Error(err)
	}
	node2.Connect(protocol + "://" + node1Addr)

	time.Sleep(2 * time.Second)

	err = CheckConnectedNodes(node1.activePeers, []string{protocol + "://" + node2Addr})
	if err != nil {
		t.Error(err)
	}

	err = CheckConnectedNodes(node2.activePeers, []string{protocol + "://" + node1Addr})
	if err != nil {
		t.Error(err)
	}
	node3.Start(tc)

	node2.Connect(protocol + "://" + node3Addr)

	time.Sleep(2 * time.Second)

	err = CheckConnectedNodes(node1.activePeers, []string{protocol + "://" + node2Addr})
	if err != nil {
		t.Error(err)
	}

	err = CheckConnectedNodes(node2.activePeers, []string{protocol + "://" + node1Addr, protocol + "://" + node3Addr})
	if err != nil {
		t.Error(err)
	}

	err = CheckConnectedNodes(node3.activePeers, []string{protocol + "://" + node2Addr})
	if err != nil {
		t.Error(err)
	}

	node1.Stop()

	err = CheckConnectedNodes(node2.activePeers, []string{protocol + "://" + node3Addr})
	if err != nil {
		t.Error(err)
	}

	err = CheckConnectedNodes(node3.activePeers, []string{protocol + "://" + node2Addr})
	if err != nil {
		t.Error(err)
	}
}
