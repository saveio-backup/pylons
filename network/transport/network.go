package transport


import (
	"fmt"
	"sync"
	"time"
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/oniio/oniP2p/network"
	"github.com/oniio/oniP2p/types/opcode"
	"github.com/oniio/oniP2p/network/keepalive"
	"github.com/oniio/oniP2p/crypto"
	"github.com/oniio/oniP2p/crypto/ed25519"
	"github.com/oniio/oniP2p/network/addressmap"
	"github.com/saveio/themis/common/log"
	"github.com/ontio/ontology-eventbus/actor"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/pylons/network/transport/messages"
)

const (
	OpCodeProcessed opcode.Opcode = 1000 + iota
	OpCodeDelivered
	OpCodeSecretRequest
	OpCodeRevealSecret
	OpCodeSecretMsg
	OpCodeDirectTransfer
	OpCodeLockedTransfer
	OpCodeRefundTransfer
	OpCodeLockExpired
	OpCodeWithdrawRequest
	OpCodeWithdraw
	OpCodeCooperativeSettleRequest
	OpCodeCooperativeSettle
)

var opCodes = map[opcode.Opcode]proto.Message{
	OpCodeProcessed:                &messages.Processed{},
	OpCodeDelivered:                &messages.Delivered{},
	OpCodeSecretRequest:            &messages.SecretRequest{},
	OpCodeRevealSecret:             &messages.RevealSecret{},
	OpCodeSecretMsg:                &messages.Secret{},
	OpCodeDirectTransfer:           &messages.DirectTransfer{},
	OpCodeLockedTransfer:           &messages.LockedTransfer{},
	OpCodeRefundTransfer:           &messages.RefundTransfer{},
	OpCodeLockExpired:              &messages.LockExpired{},
	OpCodeWithdrawRequest:          &messages.WithdrawRequest{},
	OpCodeWithdraw:                 &messages.Withdraw{},
	OpCodeCooperativeSettleRequest: &messages.CooperativeSettleRequest{},
	OpCodeCooperativeSettle:        &messages.CooperativeSettle{},
}

type P2pNetwork struct {
	*network.Component
	net *network.Network
	peerAddrs  []string
	listenAddr string
	pid *actor.PID


	protocol              string
	address               string
	mappingAddress        string
	keys                  *crypto.KeyPair
	keepaliveInterval     time.Duration
	keepaliveTimeout      time.Duration
	peerStateChan         chan *keepalive.PeerStateEvent
	kill                  chan struct{}
	activePeers           *sync.Map
	addressForHealthCheck *sync.Map
}

var once sync.Once

func NewNetWork() *P2pNetwork{
	n:=&P2pNetwork{}
	n.net = new(network.Network)
	n.activePeers = new(sync.Map)
	n.addressForHealthCheck = new(sync.Map)
	n.kill = make(chan struct{})
	n.peerStateChan = make(chan *keepalive.PeerStateEvent, 10)
	return n
}

func (this *P2pNetwork) Start() error {
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
		log.Error("[P2pNetwork] Start builder.Build error: ", err.Error())
		return err
	}

	once.Do(func() {
		for k, v := range opCodes {
			err := opcode.RegisterMessageType(k, v)
			if err != nil {
				panic("register messages failed")
			}
		}
	})

	go this.net.Listen()
	go this.syncPeerState()

	this.net.BlockUntilListening()
	return nil
}

func (this *P2pNetwork) Stop() {
	close(this.kill)
	this.net.Close()
}

func (this *P2pNetwork) syncPeerState() {
	var nodeNetworkState string
	for {
		select {
		case state := <-this.peerStateChan:
			if state.State == keepalive.PEER_REACHABLE {
				log.Debugf("[syncPeerState] addr: %s state: NetworkReachable\n", state.Address)
				this.activePeers.LoadOrStore(state.Address, struct{}{})
				nodeNetworkState = transfer.NetworkReachable
			} else {
				this.activePeers.Delete(state.Address)
				log.Debugf("[syncPeerState] addr: %s state: NetworkUnreachable\n", state.Address)
				nodeNetworkState = transfer.NetworkUnreachable
			}
			SetNodeNetworkState(state.Address, nodeNetworkState)
		case <-this.kill:
			log.Warn("[syncPeerState] this.kill...")
			break
		}
	}
}

//P2P network msg receive. transfer to actor_channel
func (this *P2pNetwork) Receive(message proto.Message, from string) error {
	log.Debug("[P2pNetwork] Receive msgBus is accepting for syncNet messages ")

	switch msg := message.(type) {
	case *messages.Processed:
		OnBusinessMessage(message, from)
	case *messages.Delivered:
		OnBusinessMessage(message, from)
	case *messages.SecretRequest:
		OnBusinessMessage(message, from)
	case *messages.RevealSecret:
		OnBusinessMessage(message, from)
	case *messages.Secret:
		OnBusinessMessage(message, from)
	case *messages.DirectTransfer:
		OnBusinessMessage(message, from)
	case *messages.LockedTransfer:
		OnBusinessMessage(message, from)
	case *messages.RefundTransfer:
		OnBusinessMessage(message, from)
	case *messages.LockExpired:
		OnBusinessMessage(message, from)
	case *messages.WithdrawRequest:
		OnBusinessMessage(message, from)
	case *messages.Withdraw:
		OnBusinessMessage(message, from)
	case *messages.CooperativeSettleRequest:
		OnBusinessMessage(message, from)
	case *messages.CooperativeSettle:
		OnBusinessMessage(message, from)

	default:
		log.Errorf("[P2pNetwork Receive] unknown message type:%s", msg.String())
	}

	return nil
}

func (this *P2pNetwork) Connect(tAddr string) error {
	if _, ok := this.activePeers.Load(tAddr); ok {
		// node is active, no need to connect
		return nil
	}
	if _, ok := this.addressForHealthCheck.Load(tAddr); ok {
		// already try to connect, donn't retry before we get a result
		return nil
	}

	this.addressForHealthCheck.Store(tAddr, struct{}{})

	log.Debug("[Connect]  To Addr: ", tAddr)
	this.net.Bootstrap(tAddr)
	if exist := this.net.ConnectionStateExists(tAddr); !exist {
		log.Errorf("[P2P connect] bootstrap addr: %s error ", tAddr)
		return fmt.Errorf("[P2P connect] bootstrap addr:%s error",tAddr)
	} else {



		log.Infof("[P2P connect] bootstrap addr: %s successfully ", tAddr)
	}

	return nil
}

func (this *P2pNetwork) Close(tAddr string) error {
	peer, err := this.net.Client(tAddr)
	if err != nil {
		log.Error("[P2P Close] close addr: %s error ", tAddr)
	} else {
		this.addressForHealthCheck.Delete(tAddr)
		peer.Close()
	}
	return nil
}

// Send send msg to peer asyncnously
// peer can be addr(string) or client(*network.peerClient)
func (this *P2pNetwork) Send(msg proto.Message, toAddr string) error {
	if _, ok := this.activePeers.Load(toAddr); !ok {
		return fmt.Errorf("can not send to inactive peer %s", toAddr)
	}
	signed, err := this.net.PrepareMessage(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("failed to sign message")
	}
	err = this.net.Write(toAddr, signed)
	if err != nil {
		return fmt.Errorf("failed to send message to %s", toAddr)
	}
	return nil
}

// SetPID sets p2p actor
func (this *P2pNetwork) SetPID(pid *actor.PID) {
	this.pid = pid
}

// GetPID returns p2p actor
func (this *P2pNetwork) GetPID() *actor.PID {
	return this.pid
}

func (this *P2pNetwork) SetProtocol(protocol string) {
	this.protocol = protocol
}

func (this *P2pNetwork) GetProtocol() string {
	return this.protocol
}

func (this *P2pNetwork) SetAddress(address string) {
	this.address = address
}

func (this *P2pNetwork) GetAddress() string {
	return this.address
}

func (this *P2pNetwork) SetMappingAddress(mappingAddress string) {
	this.mappingAddress = mappingAddress
}

func (this *P2pNetwork) GetMappingAddress() string {
	return this.mappingAddress
}

func (this *P2pNetwork) SetKeys(keys *crypto.KeyPair) {
	this.keys = keys
}

func (this *P2pNetwork) GetKeys() *crypto.KeyPair {
	return this.keys
}

func (this *P2pNetwork) GetFullAddress() string {
	if this.address != "" {
		return this.protocol + "://" + this.address
	}
	return ""
}

func (this *P2pNetwork) GetFullMappingAddress() string {
	if this.mappingAddress != "" {
		return this.protocol + "://" + this.mappingAddress
	}
	return ""
}

func (this *P2pNetwork) GetListenAddr() string {
	return this.listenAddr
}

func (this *P2pNetwork) GetPeersIfExist() error {
	this.net.EachPeer(func(client *network.PeerClient) bool {
		this.peerAddrs = append(this.peerAddrs, client.Address)
		return true
	})
	return nil
}


