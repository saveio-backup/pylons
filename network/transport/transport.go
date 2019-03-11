package transport

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"reflect"

	"github.com/gogo/protobuf/proto"
	lru "github.com/hashicorp/golang-lru"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/common/constants"
	"github.com/oniio/oniChannel/network/transport/messages"
	"github.com/oniio/oniChannel/transfer"
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
	HandleStateChange(stateChange transfer.StateChange) []transfer.Event
	Get(nodeAddress common.Address) string
	StateFromChannel() *transfer.ChainState
}

type Discoverer interface {
	Get(nodeAddress common.Address) string
}

type Transport struct {
	net *network.Network

	//protocol could be Tcp, Kcp
	protocol               string
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
	// messsage handler and signer, reference channel service
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

func NewTransport(protocol string) *Transport {
	return &Transport{
		protocol:              protocol,
		peerStateChan:         make(chan *keepalive.PeerStateEvent, 10),
		activePeers:           new(sync.Map),
		addressForHealthCheck: new(sync.Map),
		kill:                  make(chan struct{}),
		messageQueues:         new(sync.Map),
		addressQueueMap:       new(sync.Map),
		hostPortToAddress:     new(sync.Map),
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
	//rec := chainComm.Address(queueId.Recipient)
	//log.Debug("[SendAsync] %v, TO: %v.", reflect.TypeOf(msg).String(), rec.ToBase58())
	q := this.GetQueue(queueId)
	switch msg.(type) {
	case *messages.DirectTransfer:
		msgID = (msg.(*messages.DirectTransfer)).MessageIdentifier
	case *messages.Processed:
		msgID = (msg.(*messages.Processed)).MessageIdentifier
	case *messages.LockedTransfer:
		msgID = (msg.(*messages.LockedTransfer)).BaseMessage.MessageIdentifier
	case *messages.SecretRequest:
		msgID = (msg.(*messages.SecretRequest)).MessageIdentifier
	case *messages.RevealSecret:
		msgID = (msg.(*messages.RevealSecret)).MessageIdentifier
	case *messages.Secret:
		msgID = (msg.(*messages.Secret)).MessageIdentifier
	case *messages.LockExpired:
		msgID = (msg.(*messages.LockExpired)).MessageIdentifier
	default:
		log.Error("[SendAsync] Unknown message type to send async: ", reflect.TypeOf(msg).String())
		return fmt.Errorf("Unknown message type to send async")
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
	q := NewQueue(constants.MAX_MSG_QUEUE)

	this.messageQueues.Store(*queueId, q)

	go this.QueueSend(q, queueId)

	return q
}

func (this *Transport) QueueSend(queue *Queue, queueId *transfer.QueueIdentifier) {
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
			err := this.PeekAndSend(queue, queueId)
			if err != nil {
				log.Error("send message failed:", err)
				t.Stop()
				break
			}
		case msgId := <-queue.DeliverChan:

			data, _ := queue.Peek()
			if data == nil {
				continue
			}
			item := data.(*QueueItem)
			fmt.Printf("msgId = %+v\n", msgId.MessageId)
			fmt.Printf("item = %+v\n", item.messageId)
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
	fmt.Printf("send msg msg = %+v\n", msg)
	address := this.GetHostPortFromAddress(queueId.Recipient)
	if address == "" {
		return errors.New("no valid address to send message")
	}
	this.addressQueueMap.LoadOrStore(address, queue)
	err := this.Send(address, msg)
	if err != nil {
		return err
	}

	return nil
}

func (this *Transport) GetAddressCacheValue(address common.Address) string {
	if this.addressToHostPortCache == nil {
		return ""
	}
	hostPort, ok := this.addressToHostPortCache.Get(address)
	if ok {
		return hostPort.(string)
	}
	return ""
}

func (this *Transport) SaveAddressCache(address common.Address, hostPort string) bool {
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

func (this *Transport) GetHostPortFromAddress(recipient common.Address) string {
	hostPort := this.GetAddressCacheValue(recipient)
	if hostPort == "" {
		hostPort = this.ChannelService.Get(recipient)
		if hostPort == "" {
			log.Error("can`t get host and port of reg address")
			return ""
		}
		this.SaveAddressCache(recipient, hostPort)
	}

	address := hostPort
	return address
}

func (this *Transport) StartHealthCheck(address common.Address) {
	// transport not started yet, dont try to connect
	if this.net == nil {
		return
	}

	nodeAddress := this.GetHostPortFromAddress(address)
	if nodeAddress == "" {
		log.Error("node address invalid,can`t check health")
		return
	}
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
	//this.SetNodeNetworkState(address, transfer.NetworkUnreachable) //default value before connect
	this.Connect(nodeAddress)
}

func (this *Transport) SetNodeNetworkState(address common.Address, state string) {
	chainState := this.ChannelService.StateFromChannel()
	if chainState != nil {
		chainState.NodeAddressesToNetworkstates[address] = state
	}
}

func (this *Transport) Receive(message proto.Message, from string) {
	log.Debug("[NetComponent] Receive: ", reflect.TypeOf(message).String(), " From: ", from)
	switch message.(type) {
	case *messages.Delivered:
		go this.ReceiveDelivered(message, from)
	default:
		go this.ReceiveMessage(message, from)
	}
}

func (this *Transport) ReceiveMessage(message proto.Message, from string) {
	log.Debug("[ReceiveMessage] %v from: %v", reflect.TypeOf(message).String(), from)

	if this.ChannelService != nil {
		this.ChannelService.OnMessage(message, from)
	}
	//log.Info("ReceiveMessage")

	var address common.Address
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
	case *messages.LockedTransfer:
		msg := message.(*messages.LockedTransfer)
		address = messages.ConvertAddress(msg.BaseMessage.EnvelopeMessage.Signature.Sender)
		msgID = msg.BaseMessage.MessageIdentifier
	case *messages.SecretRequest:
		msg := message.(*messages.SecretRequest)
		address = messages.ConvertAddress(msg.Signature.Sender)
		msgID = msg.MessageIdentifier
	case *messages.RevealSecret:
		msg := message.(*messages.RevealSecret)
		address = messages.ConvertAddress(msg.Signature.Sender)
		msgID = msg.MessageIdentifier
	case *messages.Secret:
		msg := message.(*messages.Secret)
		address = messages.ConvertAddress(msg.EnvelopeMessage.Signature.Sender)
		msgID = msg.MessageIdentifier
	default:
		log.Warn("[ReceiveMessage] unkown Msg type: ", reflect.TypeOf(message).String())
	}

	deliveredMessage := &messages.Delivered{
		DeliveredMessageIdentifier: msgID,
	}

	var nodeAddress string
	err := this.ChannelService.Sign(deliveredMessage)
	if err == nil {
		if address != common.EmptyAddress {
			nodeAddress = this.GetHostPortFromAddress(address)
		} else {
			nodeAddress = this.protocol + "://" + from
		}
		log.Debugf("[ReceiveMessage] Send (%v) deliveredMessage from: %v",
			reflect.TypeOf(message).String(), nodeAddress)
		this.Send(nodeAddress, deliveredMessage)
	} else {
		log.Errorf("[ReceiveMessage] (%v) deliveredMessage Sign error: ", err.Error(),
			reflect.TypeOf(message).String(), nodeAddress)
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
	//log.Info("ReceiveDelivered for", msg.DeliveredMessageIdentifier)
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
				nodeNetworkState = transfer.NetworkReachable
			} else {
				this.activePeers.Delete(state.Address)
				nodeNetworkState = transfer.NetworkUnreachable
			}

			this.addressForHealthCheck.Delete(state.Address)
			address, ok := this.hostPortToAddress.Load(state.Address)

			if !ok {
				continue
			}
			this.SetNodeNetworkState(address.(common.Address), nodeNetworkState)
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

func (this *Transport) Start(channelservice ChannelServiceInterface) error {

	this.ChannelService = channelservice

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
	var err error
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
