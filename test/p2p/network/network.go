package network

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/ontio/ontology-eventbus/actor"
	"github.com/saveio/carrier/crypto"
	"github.com/saveio/carrier/crypto/ed25519"
	"github.com/saveio/carrier/network"
	"github.com/saveio/carrier/network/components/backoff"
	"github.com/saveio/carrier/network/components/keepalive"
	"github.com/saveio/carrier/network/components/proxy"
	"github.com/saveio/carrier/types/opcode"
	"github.com/saveio/dsp-go-sdk/network/common"
	"github.com/saveio/dsp-go-sdk/network/message/pb"
	dspmsg "github.com/saveio/dsp-go-sdk/network/message/pb"
	act "github.com/saveio/pylons/actor/server"
	"github.com/saveio/pylons/network/transport/messages"
	"github.com/saveio/pylons/test/media/test_config"
	pm "github.com/saveio/scan/messages/protoMessages"
	"github.com/saveio/themis/common/log"
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

// TODO: remove this
const (
	TempOpCodeRegistry   = 1013
	TempOpCodeUnRegistry = 1014
	TempOpCodeTorrent    = 1015
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
	TempOpCodeTorrent:              &pm.Torrent{}, // TODO: remove this after scan refactoring
	TempOpCodeRegistry:             &pm.Registry{},
	TempOpCodeUnRegistry:           &pm.UnRegistry{},
}

type Network struct {
	P2p                   *network.Network
	peerAddrs             []string
	listenAddr            string
	proxyAddr             string
	pid                   *actor.PID
	protocol              string
	address               string
	mappingAddress        string
	Keys                  *crypto.KeyPair
	keepaliveInterval     time.Duration
	keepaliveTimeout      time.Duration
	peerStateChan         chan *keepalive.PeerStateEvent
	kill                  chan struct{}
	addressForHealthCheck *sync.Map
	handler               func(*network.ComponentContext)
	backOff               *backoff.Component
	keepalive             *keepalive.Component
}

func NewP2P() *Network {
	n := &Network{
		P2p: new(network.Network),
	}
	n.addressForHealthCheck = new(sync.Map)
	n.kill = make(chan struct{})
	n.peerStateChan = make(chan *keepalive.PeerStateEvent, 100)
	return n

}

func (this *Network) SetProxyServer(address string) {
	this.proxyAddr = address
}

func (this *Network) GetProxyServer() string {
	return this.proxyAddr
}

func (this *Network) SetNetworkKey(key *crypto.KeyPair) {
	this.Keys = key
}

func (this *Network) SetHandler(handler func(*network.ComponentContext)) {
	this.handler = handler
}

func (this *Network) Protocol() string {
	return getProtocolFromAddr(this.PublicAddr())
}

func (this *Network) ListenAddr() string {
	return this.listenAddr
}

func (this *Network) PublicAddr() string {
	return this.P2p.ID.Address
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
}

// GetPID returns p2p actor
func (this *Network) GetPID() *actor.PID {
	return this.pid
}

func (this *Network) Start(address string) error {
	this.protocol = getProtocolFromAddr(address)
	log.Infof("Network protocol %s addr: %s", this.protocol, address)
	builder := network.NewBuilderWithOptions(network.WriteFlushLatency(1 * time.Millisecond))

	// set keys
	if this.Keys != nil {
		log.Debugf("Network use account key")
		builder.SetKeys(this.Keys)
	} else {
		log.Debugf("Network use RandomKeyPair key")
		builder.SetKeys(ed25519.RandomKeyPair())
	}

	builder.SetAddress(address)
	// add msg receiver
	component := new(NetComponent)
	component.Net = this
	builder.AddComponent(component)

	// add keepalive
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
	this.keepalive = keepalive.New(options...)
	builder.AddComponent(this.keepalive)

	// add backoff
	backoff_options := []backoff.ComponentOption{
		backoff.WithInitialDelay(3 * time.Second),
		backoff.WithMaxAttempts(10),
		backoff.WithPriority(65535),
	}
	this.backOff = backoff.New(backoff_options...)
	builder.AddComponent(this.backOff)
	this.AddProxyComponents(builder)

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
				log.Errorf("register messages failed %v", k)
			}
		}
		opcode.RegisterMessageType(opcode.Opcode(common.MSG_OP_CODE), &pb.Message{})
	})

	if len(test_config.Parameters.BaseConfig.NATProxyServerAddrs) > 0 {
		this.P2p.EnableProxyMode(true)
		this.P2p.SetProxyServer(test_config.Parameters.BaseConfig.NATProxyServerAddrs)
	}

	log.Infof("network Id : %d", test_config.Parameters.BaseConfig.NetworkId)
	this.P2p.SetNetworkID(test_config.Parameters.BaseConfig.NetworkId)
	go this.P2p.Listen()
	// go this.PeerStateChange(this.syncPeerState)
	this.P2p.BlockUntilListening()
	err = this.StartProxy(builder)
	if err != nil {
		return err
	}
	if len(this.P2p.ID.Address) == 6 {
		return errors.New("invalid address")
	}
	return nil
}

func (this *Network) AddProxyComponents(builder *network.Builder) {
	servers := strings.Split(test_config.Parameters.BaseConfig.NATProxyServerAddrs, ",")
	hasAdd := make(map[string]struct{})
	for _, proxyAddr := range servers {
		if len(proxyAddr) == 0 {
			continue
		}
		protocol := getProtocolFromAddr(proxyAddr)
		_, ok := hasAdd[protocol]
		if ok {
			continue
		}
		switch protocol {
		case "udp":
			builder.AddComponent(new(proxy.UDPProxyComponent))
		case "kcp":
			builder.AddComponent(new(proxy.KCPProxyComponent))
		case "quic":
			builder.AddComponent(new(proxy.QuicProxyComponent))
		case "tcp":
			builder.AddComponent(new(proxy.TcpProxyComponent))
		}
		hasAdd[protocol] = struct{}{}
	}
}

func (this *Network) StartProxy(builder *network.Builder) error {
	var err error
	log.Debugf("NATProxyServerAddrs :%v", test_config.Parameters.BaseConfig.NATProxyServerAddrs)
	servers := strings.Split(test_config.Parameters.BaseConfig.NATProxyServerAddrs, ",")
	for _, proxyAddr := range servers {
		if len(proxyAddr) == 0 {
			continue
		}
		this.P2p.EnableProxyMode(true)
		this.P2p.SetProxyServer(proxyAddr)
		protocol := getProtocolFromAddr(proxyAddr)
		log.Debugf("start proxy will blocking...%s %s, networkId: %d", protocol, proxyAddr, test_config.Parameters.BaseConfig.NetworkId)
		done := make(chan struct{}, 1)
		go func() {
			switch protocol {
			case "udp":
				this.P2p.BlockUntilUDPProxyFinish()
			case "kcp":
				this.P2p.BlockUntilKCPProxyFinish()
			case "quic":
				this.P2p.BlockUntilQuicProxyFinish()
			case "tcp":
				this.P2p.BlockUntilTcpProxyFinish()
			}
			done <- struct{}{}
		}()
		select {
		case <-done:
			this.proxyAddr = proxyAddr
			log.Debugf("start proxy finish, publicAddr: %s", this.P2p.ID.Address)
			return nil
		case <-time.After(time.Minute):
			err = fmt.Errorf("proxy: %s timeout", proxyAddr)
			log.Debugf("start proxy err :%s", err)
			break
		}
	}
	return err
}

func (this *Network) Stop() {
	close(this.kill)
	this.P2p.Close()
}

func (this *Network) Connect(tAddr string) error {
	if this == nil {
		return fmt.Errorf("[Connect] this is nil")
	}
	peerState, _ := this.GetPeerStateByAddress(tAddr)
	if peerState == keepalive.PEER_REACHABLE {
		return nil
	}

	if _, ok := this.addressForHealthCheck.Load(tAddr); ok {
		// already try to connect, don't retry before we get a result
		log.Info("already try to connect")
		return nil
	}

	this.addressForHealthCheck.Store(tAddr, struct{}{})
	this.P2p.Bootstrap(tAddr)
	return nil
}

func (this *Network) ConnectAndWait(addr string) error {
	if this.IsConnectionExists(addr) {
		log.Debugf("connection exist %s", addr)
		return nil
	}
	log.Debugf("bootstrap to %v", addr)
	this.P2p.Bootstrap(addr)
	err := this.WaitForConnected(addr, time.Duration(5)*time.Second)
	if err != nil {
		return err
	}
	return nil
}

func (this *Network) GetPeerStateByAddress(addr string) (keepalive.PeerState, error) {
	return this.keepalive.GetPeerStateByAddress(addr)
}

func (this *Network) WaitForConnected(addr string, timeout time.Duration) error {
	interval := time.Duration(1) * time.Second
	secs := int(timeout / interval)
	if secs <= 0 {
		secs = 1
	}
	for i := 0; i < secs; i++ {
		state, _ := this.GetPeerStateByAddress(addr)
		log.Debugf("this.keepalive: %p, GetPeerStateByAddress state addr:%s, :%d", this.keepalive, addr, state)
		if state == keepalive.PEER_REACHABLE {
			return nil
		}
		<-time.After(interval)
	}
	return errors.New("wait for connected timeout")
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

func (this *Network) Dial(addr string) error {
	log.Debugf("[DSPNetwork] Dial")
	if this.P2p == nil {
		return errors.New("network is nil")
	}
	_, err := this.P2p.Dial(addr)
	return err
}

func (this *Network) Disconnect(addr string) error {
	if this.P2p == nil {
		return errors.New("network is nil")
	}
	peer := this.P2p.GetPeerClient(addr)
	if peer == nil {
		return errors.New("client is nil")
	}
	return peer.Close()
}

func (this *Network) IsConnectionExists(addr string) bool {
	if this.P2p == nil {
		return false
	}
	return this.P2p.ConnectionStateExists(addr)
}

// Send send msg to peer asynchronous
// peer can be addr(string) or client(*network.peerClient)
func (this *Network) Send(msg proto.Message, toAddr string) error {
	// if _, ok := this.ActivePeers.Load(toAddr); !ok {
	// 	return fmt.Errorf("can not send to inactive peer %s", toAddr)
	// }
	state, _ := this.keepalive.GetPeerStateByAddress(toAddr)
	if state != keepalive.PEER_REACHABLE {
		return fmt.Errorf("can not send to inactive peer %s", toAddr)
	}
	signed, err := this.P2p.PrepareMessage(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("failed to sign message")
	}
	log.Debugf("write msg to %s", toAddr)
	err = this.P2p.Write(toAddr, signed)
	log.Debugf("write msg done to %s", toAddr)
	if err != nil {
		return fmt.Errorf("failed to send message to %s", toAddr)
	}
	return nil
}

// Request. send msg to peer and wait for response synchronously with timeout
func (this *Network) Request(msg proto.Message, peer string) (proto.Message, error) {
	client := this.P2p.GetPeerClient(peer)
	if client == nil {
		return nil, fmt.Errorf("get peer client is nil %s", peer)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(common.REQUEST_MSG_TIMEOUT)*time.Second)
	defer cancel()
	return client.Request(ctx, msg)
}

// RequestWithRetry. send msg to peer and wait for response synchronously
func (this *Network) RequestWithRetry(msg proto.Message, peer string, retry int) (proto.Message, error) {
	client := this.P2p.GetPeerClient(peer)
	if client == nil {
		return nil, fmt.Errorf("get peer client is nil %s", peer)
	}
	var res proto.Message
	var err error
	for i := 0; i < retry; i++ {
		log.Debugf("send request msg to %s with retry %d", peer, i)
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(common.REQUEST_MSG_TIMEOUT)*time.Second)
		defer cancel()
		res, err = client.Request(ctx, msg)
		if err == nil || err.Error() != "context deadline exceeded" {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Broadcast. broadcast same msg to peers. Handle action if send msg success.
// If one msg is sent failed, return err. But the previous success msgs can not be recalled.
// callback(responseMsg, responseToAddr).
func (this *Network) Broadcast(addrs []string, msg proto.Message, needReply bool, stop func() bool, callback func(proto.Message, string)) (map[string]error, error) {
	log.Debugf("[DSPNetwork] broadcast")
	maxRoutines := common.MAX_GOROUTINES_IN_LOOP
	if len(addrs) <= common.MAX_GOROUTINES_IN_LOOP {
		maxRoutines = len(addrs)
	}
	type broadcastReq struct {
		addr string
	}
	type broadcastResp struct {
		addr string
		err  error
	}
	dispatch := make(chan *broadcastReq, 0)
	done := make(chan *broadcastResp, 0)
	for i := 0; i < maxRoutines; i++ {
		go func() {
			for {
				req, ok := <-dispatch
				log.Debugf("receive request from %s, ok %t", req, ok)
				if !ok || req == nil {
					break
				}
				if !this.IsConnectionExists(req.addr) {
					log.Debugf("broadcast msg check %v not exist, connecting...", req.addr)
					err := this.ConnectAndWait(req.addr)
					if err != nil {
						done <- &broadcastResp{
							addr: req.addr,
							err:  err,
						}
						continue
					}
				}
				var res proto.Message
				var err error
				if !needReply {
					err = this.Send(msg, req.addr)
				} else {
					res, err = this.Request(msg, req.addr)
				}
				if err != nil {
					log.Errorf("broadcast msg to %s err %s", req.addr, err)
					done <- &broadcastResp{
						addr: req.addr,
						err:  err,
					}
					continue
				}
				if callback != nil {
					callback(res, req.addr)
				}
				done <- &broadcastResp{
					addr: req.addr,
					err:  nil,
				}
			}
		}()
	}
	go func() {
		for _, addr := range addrs {
			dispatch <- &broadcastReq{
				addr: addr,
			}
		}
	}()

	m := make(map[string]error)
	for {
		result := <-done
		m[result.addr] = result.err
		if stop != nil && stop() {
			return m, nil
		}
		if len(m) != len(addrs) {
			continue
		}
		close(dispatch)
		close(done)
		return m, nil
	}
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

//P2P network msg receive. transfer to actor_channel
func (this *Network) Receive(ctx *network.ComponentContext, message proto.Message, from string) error {
	// TODO check message is nil
	switch message.(type) {
	case *messages.Processed:
		act.OnBusinessMessage(message, from)
	case *messages.Delivered:
		act.OnBusinessMessage(message, from)
	case *messages.SecretRequest:
		act.OnBusinessMessage(message, from)
	case *messages.RevealSecret:
		act.OnBusinessMessage(message, from)
	case *messages.Secret:
		act.OnBusinessMessage(message, from)
	case *messages.DirectTransfer:
		act.OnBusinessMessage(message, from)
	case *messages.LockedTransfer:
		act.OnBusinessMessage(message, from)
	case *messages.RefundTransfer:
		act.OnBusinessMessage(message, from)
	case *messages.LockExpired:
		act.OnBusinessMessage(message, from)
	case *messages.WithdrawRequest:
		act.OnBusinessMessage(message, from)
	case *messages.Withdraw:
		act.OnBusinessMessage(message, from)
	case *messages.CooperativeSettleRequest:
		act.OnBusinessMessage(message, from)
	case *messages.CooperativeSettle:
		act.OnBusinessMessage(message, from)
	case *dspmsg.Message:
		if this.handler != nil {
			this.handler(ctx)
		}
	default:
		// if len(msg.String()) == 0 {
		// 	log.Warnf("Receive keepalive/keepresponse msg from %s", from)
		// 	return nil
		// }
		// log.Errorf("[P2pNetwork Receive] unknown message type:%s type %T", msg.String(), message)
	}

	return nil
}

func getProtocolFromAddr(addr string) string {
	idx := strings.Index(addr, "://")
	if idx == -1 {
		return "tcp"
	}
	return addr[:idx]
}
