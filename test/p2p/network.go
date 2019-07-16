package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/ontio/ontology-eventbus/actor"
	"github.com/saveio/carrier/crypto"
	"github.com/saveio/carrier/crypto/ed25519"
	"github.com/saveio/carrier/network"
	"github.com/saveio/carrier/network/components/keepalive"
	"github.com/saveio/carrier/types/opcode"
	act "github.com/saveio/pylons/actor/server"
	"github.com/saveio/pylons/network/transport/messages"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis/common/log"
	"reflect"
)

var once sync.Once

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

type Network struct {
	P2p                   *network.Network
	peerAddrs             []string
	listenAddr            string
	pid                   *actor.PID
	protocol              string
	address               string
	mappingAddress        string
	Keys                  *crypto.KeyPair
	keepaliveInterval     time.Duration
	keepaliveTimeout      time.Duration
	peerStateChan         chan *keepalive.PeerStateEvent
	kill                  chan struct{}
	ActivePeers           *sync.Map
	addressForHealthCheck *sync.Map
}

func NewP2P() *Network {
	n := &Network{
		P2p: new(network.Network),
	}
	n.ActivePeers = new(sync.Map)
	n.addressForHealthCheck = new(sync.Map)
	n.kill = make(chan struct{})
	n.peerStateChan = make(chan *keepalive.PeerStateEvent, 16)
	return n

}

func (this *Network) Start(address string) error {
	builder := network.NewBuilderWithOptions(network.WriteFlushLatency(1 * time.Millisecond))
	if this.Keys != nil {
		builder.SetKeys(this.Keys)
	} else {
		builder.SetKeys(ed25519.RandomKeyPair())
	}

	builder.SetAddress(address)

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
	component := new(NetComponent)
	component.Net = this
	builder.AddComponent(component)
	builder.AddComponent(keepalive.New(options...))

	var err error
	this.P2p, err = builder.Build()
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

	go this.P2p.Listen()
	go this.PeerStateChange(this.syncPeerState)

	this.P2p.BlockUntilListening()

	return nil
}

func (this *Network) Stop() {
	close(this.kill)
	this.P2p.Close()
}

func (this *Network) Connect(tAddr string) error {
	if _, ok := this.ActivePeers.Load(tAddr); ok {
		// node is active, no need to connect
		pse := &keepalive.PeerStateEvent{
			Address: tAddr,
			State:   keepalive.PEER_REACHABLE,
		}
		this.peerStateChan <- pse
		return nil
	}
	if _, ok := this.addressForHealthCheck.Load(tAddr); ok {
		// already try to connect, donn't retry before we get a result
		log.Info("already try to connect")
		return nil
	}

	this.addressForHealthCheck.Store(tAddr, struct{}{})
	this.P2p.Bootstrap(tAddr)
	return nil
}

func (this *Network) Close(tAddr string) error {
	peer, err := this.P2p.Client(tAddr)
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
func (this *Network) Send(msg proto.Message, toAddr string) error {
	if _, ok := this.ActivePeers.Load(toAddr); !ok {
		return fmt.Errorf("can not send to inactive peer %s", toAddr)
	}
	ctx := network.WithSignMessage(context.Background(), false)
	signed, err := this.P2p.PrepareMessage(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to sign message")
	}
	err = this.P2p.Write(toAddr, signed)
	if err != nil {
		return fmt.Errorf("failed to send message to %s", toAddr)
	}
	return nil
}
func (this *Network) ListenAddr() string {
	return this.listenAddr
}

func (this *Network) GetPeersIfExist() error {
	this.P2p.EachPeer(func(client *network.PeerClient) bool {
		this.peerAddrs = append(this.peerAddrs, client.Address)
		return true
	})
	return nil
}

// SetPID sets p2p actor
func (this *Network) SetPID(pid *actor.PID) {
	this.pid = pid
	//this.msgRouter.SetPID(pid)
}

// GetPID returns p2p actor
func (this *Network) GetPID() *actor.PID {
	return this.pid
}

func (this *Network) PeerStateChange(fn func(*keepalive.PeerStateEvent)) {
	ka, reg := this.P2p.Component(keepalive.ComponentID)
	if !reg {
		log.Error("keepalive component do not reg")
		return
	}
	peerStateChan := ka.(*keepalive.Component).GetPeerStateChan()
	stopCh := ka.(*keepalive.Component).GetStopChan()
	for {
		select {
		case event := <-peerStateChan:
			fn(event)

		case <-stopCh:
			return

		}
	}
}
func (this *Network) syncPeerState(state *keepalive.PeerStateEvent) {
	var nodeNetworkState string
	if state.State == keepalive.PEER_REACHABLE {
		log.Debugf("[syncPeerState] addr: %s state: NetworkReachable\n", state.Address)
		this.ActivePeers.LoadOrStore(state.Address, struct{}{})
		nodeNetworkState = transfer.NetworkReachable
	} else {
		this.ActivePeers.Delete(state.Address)
		log.Debugf("[syncPeerState] addr: %s state: NetworkUnreachable\n", state.Address)
		nodeNetworkState = transfer.NetworkUnreachable
	}
	act.SetNodeNetworkState(state.Address, nodeNetworkState)
}

//P2P network msg receive. transfer to actor_channel
func (this *Network) Receive(message proto.Message, from string) error {
	log.Debug("[P2pNetwork] Receive msgBus is accepting for syncNet messages ")
	var err error
	switch msg := message.(type) {
	case *messages.Processed:
		err = act.OnBusinessMessage(message, from)
	case *messages.Delivered:
		err = act.OnBusinessMessage(message, from)
	case *messages.SecretRequest:
		err = act.OnBusinessMessage(message, from)
	case *messages.RevealSecret:
		err = act.OnBusinessMessage(message, from)
	case *messages.Secret:
		err = act.OnBusinessMessage(message, from)
	case *messages.DirectTransfer:
		err = act.OnBusinessMessage(message, from)
	case *messages.LockedTransfer:
		err = act.OnBusinessMessage(message, from)
	case *messages.RefundTransfer:
		err = act.OnBusinessMessage(message, from)
	case *messages.LockExpired:
		err = act.OnBusinessMessage(message, from)
	case *messages.WithdrawRequest:
		err = act.OnBusinessMessage(message, from)
	case *messages.Withdraw:
		err = act.OnBusinessMessage(message, from)
	case *messages.CooperativeSettleRequest:
		err = act.OnBusinessMessage(message, from)
	case *messages.CooperativeSettle:
		err = act.OnBusinessMessage(message, from)

	default:
		log.Errorf("[P2pNetwork Receive] unknown message type:%s", msg.String())
	}
	if err != nil {
		log.Errorf("[P2pNetwork Receive] Type %s OnBusinessMessage error: ", reflect.TypeOf(message).String(),
			err.Error())
	}

	return nil
}
