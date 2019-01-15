package transport

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/network"
	"github.com/oniio/oniChannel/network/transport/messages"
)

var (
	node1Addr = "tcp://127.0.0.1:3001"
	node2Addr = "tcp://127.0.0.1:3002"
	node3Addr = "tcp://127.0.0.1:3003"
)

var defaultHandler = (MessageHandler)(nil)

type TestMsgHandler struct {
	sync.Mutex
	counter   int
	transport *Transport
}

var node1AccountAddress = [20]byte{1}
var node2AccountAddress = [20]byte{2}
var node3AccountAddress = [20]byte{3}

var addressToIPMap = map[[20]byte]string{
	node1AccountAddress: "127.0.0.1:3001",
	node2AccountAddress: "127.0.0.1:3002",
	node3AccountAddress: "127.0.0.1:3003",
}

func (t *TestMsgHandler) UpdateMessageCounter() {
	t.Lock()
	t.counter++
	t.Unlock()
}

func (t *TestMsgHandler) GetMessageCounter() int {
	t.Lock()
	defer t.Unlock()
	return t.counter
}

func (t *TestMsgHandler) OnMessage(message proto.Message, from string) {
	switch message.(type) {
	case *messages.Delivered:
		return
	case *messages.DirectTransfer:
		t.UpdateMessageCounter()

		msg := message.(*messages.DirectTransfer)
		deliverMsg := &messages.Delivered{
			DeliveredMessageIdentifier: &messages.MessageID{msg.MessageIdentifier.MessageId},
		}

		t.transport.Send(from, deliverMsg)
	}
}

type TestDiscoverer struct{}

func (t *TestDiscoverer) Get(nodeAddress common.Address) string {
	address, ok := addressToIPMap[nodeAddress]
	if !ok {
		return ""
	}

	return address
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
	node1 := NewTransport("tcp")
	node2 := NewTransport("tcp")
	node3 := NewTransport("tcp")

	node1.SetAddress(node1Addr)
	node2.SetAddress(node2Addr)
	node3.SetAddress(node3Addr)
	bs := network.BlockchainService{}
	node1.Start(bs)
	node2.Start(bs)

	node2.Connect(node1Addr)

	time.Sleep(2 * time.Second)

	err = CheckConnectedNodes(node1.activePeers, []string{node2Addr})
	if err != nil {
		t.Error(err)
	}

	err = CheckConnectedNodes(node2.activePeers, []string{node1Addr})
	if err != nil {
		t.Error(err)
	}
	node3.Start(bs)

	node2.Connect(node3Addr)

	time.Sleep(2 * time.Second)

	err = CheckConnectedNodes(node1.activePeers, []string{node2Addr})
	if err != nil {
		t.Error(err)
	}

	err = CheckConnectedNodes(node2.activePeers, []string{node1Addr, node3Addr})
	if err != nil {
		t.Error(err)
	}

	err = CheckConnectedNodes(node3.activePeers, []string{node2Addr})
	if err != nil {
		t.Error(err)
	}

	node1.Close()

	err = CheckConnectedNodes(node2.activePeers, []string{node3Addr})
	if err != nil {
		t.Error(err)
	}

	err = CheckConnectedNodes(node3.activePeers, []string{node2Addr})
	if err != nil {
		t.Error(err)
	}
}

func TestQueueSend(t *testing.T) {

	msgHandler1 := new(TestMsgHandler)
	msgHandler1.transport = NewTransport("tcp")

	msgHandler2 := new(TestMsgHandler)
	msgHandler2.transport = NewTransport("tcp")

	msgHandler3 := new(TestMsgHandler)
	msgHandler3.transport = NewTransport("tcp")

	node1 := msgHandler1.transport
	node2 := msgHandler2.transport
	node3 := msgHandler3.transport

	node1.SetAddress(node1Addr)
	node2.SetAddress(node2Addr)
	node3.SetAddress(node3Addr)

	node1.Start()
	node2.Start()
	node3.Start()

	node1.Connect(node2Addr)
	node1.Connect(node3Addr)

	time.Sleep(2 * time.Second)

	var wg sync.WaitGroup

	msgID := 123
	channelID := 234
	count := 20 // NOTE: current queue length is fixed to 20

	go func() {
		wg.Add(1)
		SendQueueMessages(node1, node2AccountAddress, msgID, channelID, count)
		wg.Done()
	}()

	go func() {
		wg.Add(1)
		SendQueueMessages(node1, node3AccountAddress, msgID+100, channelID+1, count)
		wg.Done()
	}()

	wg.Wait()
	time.Sleep(2 * time.Second)

	counter := msgHandler2.GetMessageCounter()
	if counter != count {
		t.Errorf("receive counter error, expect: %d, received : %d", count, counter)
	}

	counter = msgHandler3.GetMessageCounter()
	if counter != count {
		t.Errorf("receive counter error, expect: %d, received : %d", count, counter)
	}
}

func SendQueueMessages(sender *Transport, recipient common.Address, initMessageID int, channelID int, messageCount int) {
	for i := 0; i < messageCount; i++ {
		msg := &messages.DirectTransfer{
			MessageIdentifier: &messages.MessageID{uint64(initMessageID + i)},
		}

		queueId := &QueueIdentifier{
			Recipient: recipient,
			ChannelID: common.ChannelID(channelID),
		}

		sender.SendAsync(queueId, msg, msg.MessageIdentifier)
	}
}
