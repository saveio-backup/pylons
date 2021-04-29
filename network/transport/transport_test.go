package transport

import (
	"fmt"
	"sync"

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
