package transport

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	lru "github.com/hashicorp/golang-lru"
	"github.com/oniio/oniChannel/network/transport/messages"
	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
	"github.com/oniio/oniP2p/crypto"
	"github.com/oniio/oniP2p/crypto/ed25519"
	"github.com/oniio/oniP2p/network"
	"github.com/oniio/oniP2p/network/addressmap"
	"github.com/oniio/oniP2p/network/keepalive"
	"github.com/oniio/oniP2p/types/opcode"
)

const ADDRESS_CACHE_SIZE = 50
const (
	OpcodeProcessed opcode.Opcode = 1000 + iota
	OpcodeDelivered
	OpcodeSecrectRequest
	OpcodeRevealSecret
	OpcodeSecrectMsg
	OpcodeDirectTransfer
	OpcodeLockedTransfer
	OpcodeRefundTransfer
	OpcodeLockExpired
)

var opcodes = map[opcode.Opcode]proto.Message{
	OpcodeProcessed:      &messages.Processed{},
	OpcodeDelivered:      &messages.Delivered{},
	OpcodeSecrectRequest: &messages.SecretRequest{},
	OpcodeRevealSecret:   &messages.RevealSecret{},
	OpcodeSecrectMsg:     &messages.Secret{},
	OpcodeDirectTransfer: &messages.DirectTransfer{},
	OpcodeLockedTransfer: &messages.LockedTransfer{},
	OpcodeRefundTransfer: &messages.RefundTransfer{},
	OpcodeLockExpired:    &messages.LockExpired{},
}

type ChannelServiceInterface interface {
	OnMessage(proto.Message, string)
	Sign(message interface{}) error
	HandleStateChange(stateChange transfer.StateChange) *list.List
}

// for test purpose
type Discoverer interface {
	Get(nodeAddress typing.Address) string
}

type Transport struct {
	net *network.Network

	//protocol could be Tcp, Kcp
	protocol               string
	discovery              Discoverer
	address                string
	mappingAddress         string
	keys                   *crypto.KeyPair
	keepaliveInterval      time.Duration
	keepaliveTimeout       time.Duration
	peerStateChan          chan *keepalive.PeerStateEvent
	activePeers            *sync.Map
	addressForHealthCheck  *sync.Map
	hostPortToAddress      *sync.Map
	addressToHostPortCache *lru.ARCCache
	// map QueueIdentifier to Queue
	messageQueues *sync.Map
	// map address to queue
	addressQueueMap *sync.Map
	kill            chan struct{}
	// messsage handler and signer, reference nimbus service
	ChannelService ChannelServiceInterface
}

type QueueItem struct {
	message   proto.Message
	messageId *messages.MessageID
}

func registerMessages() error {
	for code, msg := range opcodes {
		err := opcode.RegisterMessageType(code, msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewTransport(protocol string, discovery Discoverer) *Transport {
	return &Transport{
		protocol:              protocol,
		peerStateChan:         make(chan *keepalive.PeerStateEvent, 10),
		activePeers:           new(sync.Map),
		addressForHealthCheck: new(sync.Map),
		kill:                  make(chan struct{}),
		messageQueues:         new(sync.Map),
		addressQueueMap:       new(sync.Map),
		hostPortToAddress:     new(sync.Map),
		discovery:             discovery,
	}
}

func (this *Transport) Connect(address ...string) {
	this.net.Bootstrap(address...)
}

func (this *Transport) SetAddress(address string) {
	this.address = address
}

func (this *Transport) SetMappingAddress(mappingAddress string) {
	this.mappingAddress = mappingAddress
}

func (this *Transport) SetKeys(keys *crypto.KeyPair) {
	this.keys = keys
}

// messages first be queued, only can be send when Delivered for previous msssage is received
func (this *Transport) SendAsync(queueId *transfer.QueueIdentifier, msg proto.Message) error {
	var msgID *messages.MessageID

	q := this.GetQueue(queueId)

	switch msg.(type) {
	case *messages.DirectTransfer:
		msgID = (msg.(*messages.DirectTransfer)).MessageIdentifier
	case *messages.Processed:
		msgID = (msg.(*messages.Processed)).MessageIdentifier
	default:
		fmt.Errorf("Unknown message type to send async")
	}

	ok := q.Push(&QueueItem{
		message:   msg,
		messageId: msgID,
	})
	if !ok {
		return fmt.Errorf("failed to push to queue")
	}

	return nil
}

func (this *Transport) GetQueue(queueId *transfer.QueueIdentifier) *Queue {
	q, ok := this.messageQueues.Load(*queueId)

	if !ok {
		q = this.InitQueue(queueId)
	}

	return q.(*Queue)
}

func (this *Transport) InitQueue(queueId *transfer.QueueIdentifier) *Queue {
	q := NewQueue(1000)

	this.messageQueues.Store(*queueId, q)

	go this.QueueSend(q, queueId)

	return q
}

func (this *Transport) QueueSend(queue *Queue, queueId *transfer.QueueIdentifier) {
	//t := time.NewTicker(3 * time.Second)
	var interval time.Duration = 3

	t := time.NewTimer(interval * time.Second)

	for {
		select {
		case <-queue.DataCh:
			t.Reset(interval * time.Second)
			this.PeekAndSend(queue, queueId)
		// handle timeout retry
		case <-t.C:
			if queue.Len() == 0 {
				continue
			}

			t.Reset(interval * time.Second)
			this.PeekAndSend(queue, queueId)
		case msgId := <-queue.DeliverChan:
			data, _ := queue.Peek()
			if data == nil {
				continue
			}
			item := data.(*QueueItem)

			if msgId.MessageId == item.messageId.MessageId {
				queue.Pop()
				t.Stop()

				if queue.Len() != 0 {
					t.Reset(interval * time.Second)
					this.PeekAndSend(queue, queueId)
				}
			}
		case <-this.kill:
			t.Stop()
			break
		}
	}
}

func (this *Transport) PeekAndSend(queue *Queue, queueId *transfer.QueueIdentifier) error {
	item, ok := queue.Peek()
	if !ok {
		return fmt.Errorf("Error peeking from queue")
	}

	msg := item.(*QueueItem).message

	address := this.GetHostPortFromAddress(queueId.Recipient)
	this.addressQueueMap.LoadOrStore(address, queue)
	err := this.Send(address, msg)
	if err != nil {
		return err
	}

	return nil
}

func (this *Transport) GetAddressCacheValue(address typing.Address) string {
	if this.addressToHostPortCache == nil {
		return ""
	}
	hostPort, ok := this.addressToHostPortCache.Get(address)
	if ok {
		return hostPort.(string)
	}
	return ""
}

func (this *Transport) SaveAddressCache(address typing.Address, hostPort string) bool {
	if this.addressToHostPortCache == nil {
		var err error
		this.addressToHostPortCache, err = lru.NewARC(ADDRESS_CACHE_SIZE)
		if err != nil {
			return false
		}
	}
	this.addressToHostPortCache.Add(address, hostPort)

	//also save in the hostport to address map
	this.hostPortToAddress.Store(hostPort, address)
	return true
}

func (this *Transport) GetHostPortFromAddress(recipient typing.Address) string {
	hostPort := this.GetAddressCacheValue(recipient)
	if hostPort == "" {
		// no cache, retrive hostPort from discovery contract
		hostPort = this.discovery.Get(recipient)
		this.SaveAddressCache(recipient, hostPort)
	}

	address := this.protocol + "://" + hostPort
	return address
}

func (this *Transport) StartHealthCheck(address typing.Address) {
	// transport not started yet, dont try to connect
	if this.net == nil {
		return
	}

	nodeAddress := this.GetHostPortFromAddress(address)

	_, ok := this.activePeers.Load(nodeAddress)
	if ok {
		// node is active, no need to connect
		return
	}

	_, ok = this.addressForHealthCheck.Load(nodeAddress)
	if ok {
		// already try to connect, dont retry before we get a result
		return
	}

	this.addressForHealthCheck.Store(nodeAddress, struct{}{})
	this.Connect(nodeAddress)
}

func (this *Transport) SetNodeNetworkState(address typing.Address, state string) {
	stateChange := &transfer.ActionChangeNodeNetworkState{
		NodeAddress:  address,
		NetworkState: state,
	}

	this.ChannelService.HandleStateChange(stateChange)
}

func (this *Transport) Receive(message proto.Message, from string) {
	switch message.(type) {
	case *messages.Delivered:
		this.ReceiveDelivered(message, from)
	default:
		this.ReceiveMessage(message, from)
	}
}

func (this *Transport) ReceiveMessage(message proto.Message, from string) {
	if this.ChannelService != nil {
		this.ChannelService.OnMessage(message, from)
	}

	var address typing.Address
	var msgID *messages.MessageID

	switch message.(type) {
	case *messages.DirectTransfer:
		msg := message.(*messages.DirectTransfer)
		address = messages.ConvertAddress(msg.EnvelopeMessage.Signature.Sender)
		msgID = msg.MessageIdentifier
	case *messages.Processed:
		msg := message.(*messages.Processed)
		address = messages.ConvertAddress(msg.Signature.Sender)
		msgID = msg.MessageIdentifier
	}

	deliveredMessage := &messages.Delivered{
		DeliveredMessageIdentifier: msgID,
	}

	err := this.ChannelService.Sign(deliveredMessage)
	if err == nil {
		nodeAddress := this.GetHostPortFromAddress(address)
		this.Send(nodeAddress, deliveredMessage)
	}
}
func (this *Transport) ReceiveDelivered(message proto.Message, from string) {
	if this.ChannelService != nil {
		this.ChannelService.OnMessage(message, from)
	}

	msg := message.(*messages.Delivered)

	queue, ok := this.addressQueueMap.Load(from)
	if !ok {
		return
	}

	queue.(*Queue).DeliverChan <- msg.DeliveredMessageIdentifier
}

func (this *Transport) Send(address string, message proto.Message) error {
	if _, ok := this.activePeers.Load(address); !ok {
		return fmt.Errorf("can not send to inactive peer %s", address)
	}

	signed, err := this.net.PrepareMessage(context.Background(), message)
	if err != nil {
		return fmt.Errorf("failed to sign message")
	}

	err = this.net.Write(address, signed)
	if err != nil {
		return fmt.Errorf("failed to send message to %s", address)
	}

	return nil
}

func (this *Transport) Stop() {
	close(this.kill)
	this.net.Close()
}

func (this *Transport) syncPeerState() {
	var nodeNetworkState string
	for {
		select {
		case state := <-this.peerStateChan:
			if state.State == keepalive.PEER_REACHABLE {
				this.activePeers.LoadOrStore(state.Address, struct{}{})
				nodeNetworkState = transfer.NodeNetworkReachable
			} else {
				this.activePeers.Delete(state.Address)
				nodeNetworkState = transfer.NodeNetworkUnreachable
			}

			this.addressForHealthCheck.Delete(state.Address)

			address, ok := this.hostPortToAddress.Load(state.Address)
			if !ok {
				continue
			}
			this.SetNodeNetworkState(address.(typing.Address), nodeNetworkState)
		case <-this.kill:
			break
		}
	}
}

func (this *Transport) GetFullAddress() string {
	return this.protocol + "://" + this.address
}

func (this *Transport) GetFullMappingAddress() string {
	if this.mappingAddress != "" {
		return this.protocol + "://" + this.mappingAddress
	}

	return ""
}

var once sync.Once

func (this *Transport) Start(ChannelService ChannelServiceInterface) error {
	var err error

	this.ChannelService = ChannelService

	// must set the writeFlushLatency to proper value, otherwise the message exchange speed will be very low
	builder := network.NewBuilderWithOptions(network.WriteFlushLatency(1 * time.Millisecond))

	if this.keys != nil {
		builder.SetKeys(this.keys)
	} else {
		builder.SetKeys(ed25519.RandomKeyPair())
	}

	builder.SetAddress(this.GetFullAddress())

	component := new(NetComponent)
	component.Net = this
	builder.AddComponent(component)

	if this.mappingAddress != "" {
		builder.AddComponent(&addressmap.Component{MappingAddress: this.GetFullMappingAddress()})
	}

	if this.keepaliveInterval == 0 {
		this.keepaliveInterval = keepalive.DefaultKeepaliveInterval
	}
	if this.keepaliveTimeout == 0 {
		this.keepaliveTimeout = keepalive.DefaultKeepaliveTimeout
	}

	options := []keepalive.ComponentOption{
		keepalive.WithKeepaliveInterval(this.keepaliveInterval),
		keepalive.WithKeepaliveTimeout(this.keepaliveTimeout),
		keepalive.WithPeerStateChan(this.peerStateChan),
	}

	builder.AddComponent(keepalive.New(options...))

	this.net, err = builder.Build()
	if err != nil {
		return err
	}

	once.Do(func() {
		e := registerMessages()
		if e != nil {
			panic("register messages failed")
		}
	})

	go this.net.Listen()
	go this.syncPeerState()

	this.net.BlockUntilListening()

	return nil
}
