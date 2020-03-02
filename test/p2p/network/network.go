package network

import (
	"context"
	"errors"
	"fmt"
	"github.com/saveio/edge/p2p/peer"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/ontio/ontology-eventbus/actor"
	"github.com/saveio/carrier/crypto"
	"github.com/saveio/carrier/crypto/ed25519"
	"github.com/saveio/carrier/network"
	"github.com/saveio/carrier/network/components/ackreply"
	"github.com/saveio/carrier/network/components/keepalive"
	"github.com/saveio/carrier/network/components/keepalive/proxyKeepalive"
	//"github.com/saveio/carrier/network/components/proxy"
	"github.com/saveio/carrier/types/opcode"

	//"github.com/saveio/edge/p2p/peer"

	"github.com/saveio/pylons/actor/msg_opcode"
	act "github.com/saveio/pylons/actor/server"
	"github.com/saveio/pylons/network/transport/messages"
	"github.com/saveio/themis/common/log"
)

var once sync.Once

type Network struct {
	P2p                  *network.Network                // underlay network p2p instance
	builder              *network.Builder                // network builder
	networkId            uint32                          // network id
	intranetIP           string                          // intranet ip
	proxyAddrs           []string                        // proxy address
	pid                  *actor.PID                      // actor pid
	keys                 *crypto.KeyPair                 // crypto network keys
	kill                 chan struct{}                   // network stop signal
	handler              func(*network.ComponentContext) // network msg handler
	walletAddrFromPeerId func(string) string             // peer key from id delegate
	peers                *sync.Map                       // peer clients
	lock                 *sync.RWMutex                   // lock for sync control
	peerForHealthCheck   *sync.Map                       // peerId for keep health check
	asyncRecvDisabled    bool                            // disabled async receive msg
}

func NewP2P(opts ...NetworkOption) *Network {
	n := &Network{
		P2p: new(network.Network),
	}
	n.kill = make(chan struct{})
	n.peers = new(sync.Map)
	n.peerForHealthCheck = new(sync.Map)
	n.lock = new(sync.RWMutex)
	// update by options
	n.walletAddrFromPeerId = func(id string) string {
		return id
	}
	for _, opt := range opts {
		opt.apply(n)
	}
	return n
}

func (this *Network) WalletAddrFromPeerId(peerId string) string {
	return this.walletAddrFromPeerId(peerId)
}

// GetProxyServer. get working proxy server
func (this *Network) GetProxyServer() *network.ProxyServer {
	if this.P2p == nil {
		return &network.ProxyServer{}
	}
	addr, peerId := this.P2p.GetWorkingProxyServer()
	return &network.ProxyServer{
		IP:     addr,
		PeerID: peerId,
	}
}

// IsProxyAddr. check if the address is a proxy address
func (this *Network) IsProxyAddr(addr string) bool {
	for _, a := range this.proxyAddrs {
		if a == addr {
			return true
		}
	}
	return false
}

// Protocol. get network protocol
func (this *Network) Protocol() string {
	return getProtocolFromAddr(this.PublicAddr())
}

// PublicAddr. get network public host address
func (this *Network) PublicAddr() string {
	return this.P2p.ID.Address
}

// GetPID returns p2p actor
func (this *Network) GetPID() *actor.PID {
	return this.pid
}

// Start. start network
func (this *Network) Start(protocol, addr, port string) error {
	builderOpt := []network.BuilderOption{
		network.WriteFlushLatency(1 * time.Millisecond),
		network.WriteTimeout(8),
		// network.WriteBufferSize(common.MAX_WRITE_BUFFER_SIZE),
	}
	builder := network.NewBuilderWithOptions(builderOpt...)
	this.builder = builder

	// set keys
	if this.keys != nil {
		log.Debugf("Network use account key")
		builder.SetKeys(this.keys)
	} else {
		log.Debugf("Network use RandomKeyPair key")
		builder.SetKeys(ed25519.RandomKeyPair())
	}
	intranetHost := "127.0.0.1"
	if len(this.intranetIP) > 0 {
		intranetHost = this.intranetIP
	}

	log.Debugf("network start at %s, listen addr %s",
		fmt.Sprintf("%s://%s:%s", protocol, intranetHost, port), fmt.Sprintf("%s://%s:%s", protocol, addr, port))
	builder.SetListenAddr(fmt.Sprintf("%s://%s:%s", protocol, intranetHost, port))
	builder.SetAddress(fmt.Sprintf("%s://%s:%s", protocol, addr, port))

	// add msg receiver
	recvMsgComp := new(NetComponent)
	recvMsgComp.Net = this
	builder.AddComponent(recvMsgComp)

	// add peer component
	peerComp := new(PeerComponent)
	peerComp.Net = this
	builder.AddComponent(peerComp)

	// add peer keepalive
	options := []keepalive.ComponentOption{
		keepalive.WithKeepaliveInterval(keepalive.DefaultKeepaliveInterval),
		keepalive.WithKeepaliveTimeout(keepalive.DefaultKeepaliveTimeout),
	}
	builder.AddComponent(keepalive.New(options...))

	// add proxy keepalive
	builder.AddComponent(proxyKeepalive.New())

	// add ack reply
	ackOption := []ackreply.ComponentOption{
		ackreply.WithAckCheckedInterval(time.Second * 20),
		ackreply.WithAckMessageTimeout(time.Second * 60),
	}
	builder.AddComponent(ackreply.New(ackOption...))

	//this.addProxyComponents(builder)

	var err error
	this.P2p, err = builder.Build()
	if err != nil {
		log.Error("[P2pNetwork] Start builder.Build error: ", err.Error())
		return err
	}
	this.P2p.DisableCompress()
	if this.asyncRecvDisabled {
		this.P2p.DisableMsgGoroutine()
	}
	once.Do(func() {
		for k, v := range msg_opcode.OpCodes {
			err := opcode.RegisterMessageType(k, v)
			if err != nil {
				log.Errorf("register messages failed %v", k)
			}
		}
	})
	this.P2p.SetDialTimeout(time.Duration(5) * time.Second)
	this.P2p.SetCompressFileSize(1048576)
	// if len(this.proxyAddrs) > 0 {
	// 	this.P2p.EnableProxyMode(true)
	// 	this.P2p.SetProxyServer([]network.ProxyServer{
	// 		network.ProxyServer{
	// 			IP: this.proxyAddrs[0],
	// 		},
	// 	})
	// }
	this.P2p.SetNetworkID(this.networkId)
	go this.P2p.Listen()
	this.P2p.BlockUntilListening()
	//err = this.startProxy(builder)
	if err != nil {
		log.Errorf("start proxy failed, err: %s", err)
	}
	if len(this.P2p.ID.Address) == 6 {
		return errors.New("invalid address")
	}
	go this.healthCheckService()
	return nil
}

// Stop. stop network
func (this *Network) Stop() {
	close(this.kill)
	this.P2p.Close()
}

// GetPeerFromWalletAddr. get peer from wallet addr
func (this *Network) GetPeerFromWalletAddr(walletAddr string) *peer.Peer {
	if strings.Contains(walletAddr, "tcp") {
		log.Errorf("wallet addr is wrong %s", walletAddr)
		panic("wallet addr is wrong ")
		os.Exit(1)
	}
	p, ok := this.peers.Load(walletAddr)
	if !ok {
		return nil
	}
	pr, ok := p.(*peer.Peer)
	if pr == nil || !ok {
		return nil
	}
	return pr
}

func (this *Network) GetWalletFromHostAddr(hostAddr string) string {
	walletAddr := ""
	this.peers.Range(func(key, value interface{}) bool {
		pr, ok := value.(*peer.Peer)
		if pr == nil || !ok {
			return true
		}
		if pr.GetHostAddr() == hostAddr {
			walletAddr = key.(string)
			return false
		}
		return true
	})
	return walletAddr
}

// Connect. connect to peer with host address, store its peer id after success
func (this *Network) Connect(hostAddr string) error {
	log.Debugf("connect to host addr %s", hostAddr)
	if this == nil {
		return fmt.Errorf("network is nil")
	}
	walletAddr := this.GetWalletFromHostAddr(hostAddr)
	if len(walletAddr) > 0 {
		peerState, _ := this.GetConnStateByWallet(walletAddr)
		if peerState == network.PEER_REACHABLE {
			return nil
		}
	}
	p, ok := this.peers.Load(hostAddr)
	if ok && p.(*peer.Peer).State() == peer.ConnectStateConnecting {
		log.Debugf("already try to connect %s", hostAddr)
		return nil
	}
	var pr *peer.Peer
	if !ok {
		pr = peer.New(hostAddr)
		pr.SetState(peer.ConnectStateConnecting)
		this.peers.Store(hostAddr, pr)
	} else {
		pr = p.(*peer.Peer)
	}
	peerIds := this.P2p.Bootstrap([]string{hostAddr})
	if len(peerIds) == 0 {
		return fmt.Errorf("peer id is emptry from bootstraping to %s", hostAddr)
	}
	log.Debugf("bootstrap peerIds: %v", peerIds[0])
	pr.SetState(peer.ConnectStateConnected)
	this.peers.Delete(hostAddr)
	walletAddr = this.walletAddrFromPeerId(peerIds[0])
	// replace key with walletAddr of hostAddr
	pr.SetPeerId(peerIds[0])
	log.Debugf("connect store peerId: %s addr: %s hostAddr: %s", peerIds[0], walletAddr, hostAddr)
	this.peers.Store(walletAddr, pr)
	return nil
}

// GetPeerStateByAddress. get peer state by peerId
func (this *Network) GetConnStateByWallet(walletAddr string) (network.PeerState, error) {
	if strings.Contains(walletAddr, "tcp") {
		log.Errorf("wallet addr is wrong %s", walletAddr)
		panic("wallet addr is wrong ")
		os.Exit(1)
	}
	pr := this.GetPeerFromWalletAddr(walletAddr)
	if pr == nil {
		log.Errorf("peer not found %s", walletAddr)
		return 0, fmt.Errorf("peer not found %s", walletAddr)
	}
	peerId := pr.GetPeerId()
	s, err := this.P2p.GetRealConnState(peerId)
	if err != nil {
		return s, err
	}
	client := this.P2p.GetPeerClient(peerId)
	if client == nil {
		return s, fmt.Errorf("get peer %s client is nil", peerId)
	}
	return s, err
}

// IsConnReachable. check if peer state reachable
func (this *Network) IsConnReachable(walletAddr string) bool {
	if strings.Contains(walletAddr, "tcp") {
		log.Errorf("wallet addr is wrong %s", walletAddr)
		panic("wallet addr is wrong ")
		os.Exit(1)
	}
	if this.P2p == nil || len(walletAddr) == 0 {
		return false
	}
	pr := this.GetPeerFromWalletAddr(walletAddr)
	if pr == nil {
		return false
	}
	peerId := pr.GetPeerId()
	state, err := this.GetConnStateByWallet(peerId)
	log.Debugf("get peer state by addr: %s %v %s", peerId, state, err)
	if err != nil {
		return false
	}
	if state != network.PEER_REACHABLE {
		return false
	}
	return true
}

// WaitForConnected. poll to wait for connected
func (this *Network) WaitForConnected(walletAddr string, timeout time.Duration) error {
	if strings.Contains(walletAddr, "tcp") {
		log.Errorf("wallet addr is wrong %s", walletAddr)
		panic("wallet addr is wrong ")
		os.Exit(1)
	}
	pr := this.GetPeerFromWalletAddr(walletAddr)
	if pr == nil {
		return fmt.Errorf("peer not found %s", walletAddr)
	}
	peerId := pr.GetPeerId()
	interval := time.Duration(1) * time.Second
	secs := int(timeout / interval)
	if secs <= 0 {
		secs = 1
	}
	for i := 0; i < secs; i++ {
		state, err := this.GetConnStateByWallet(peerId)
		log.Debugf("GetPeerStateByAddress state addr:%s, :%d, err %v", peerId, state, err)
		if state == network.PEER_REACHABLE {
			return nil
		}
		<-time.After(interval)
	}
	return fmt.Errorf("wait for connecting %s timeout", peerId)
}

func (this *Network) HealthCheckPeer(walletAddr string) error {
	if strings.Contains(walletAddr, "tcp") {
		log.Errorf("wallet addr is wrong %s", walletAddr)
		panic("wallet addr is wrong ")
		os.Exit(1)
	}
	if !this.P2p.ProxyModeEnable() {
		if _, ok := this.peerForHealthCheck.Load(walletAddr); !ok {
			log.Debugf("ignore check health of peer %s, because proxy mode is disabled", walletAddr)
			return nil
		}
		log.Debugf("proxy mode is disabled, but %s exist in health check list", walletAddr)
	}
	if len(walletAddr) == 0 {
		log.Debugf("health check empty peer id")
		return nil
	}
	peerState, err := this.GetConnStateByWallet(walletAddr)
	log.Debugf("get peer state of %s, state %d, err %s", walletAddr, peerState, err)
	if peerState != network.PEER_REACHABLE {
		log.Debugf("health check peer: %s unreachable, err: %s ", walletAddr, err)
	}
	if err == nil && peerState == network.PEER_REACHABLE {
		return nil
	}
	time.Sleep(time.Duration(2) * time.Second)
	proxy := this.GetProxyServer()
	pr := this.GetPeerFromWalletAddr(walletAddr)
	if pr == nil {
		return fmt.Errorf("peer not found %s", walletAddr)
	}
	peerId := pr.GetPeerId()
	if len(proxy.IP) > 0 && peerId == proxy.PeerID {
		log.Debugf("reconnect proxy server ....")
		err = this.P2p.ReconnectProxyServer(proxy.IP, proxy.PeerID)
		if err != nil {
			log.Errorf("proxy reconnect failed, err %s", err)
			this.WaitForConnected(walletAddr, time.Duration(15)*time.Second)
			return err
		}
	} else {
		err = this.reconnect(walletAddr)
		if err != nil {
			this.WaitForConnected(walletAddr, time.Duration(15)*time.Second)
			return err
		}
	}
	err = this.WaitForConnected(walletAddr, time.Duration(15)*time.Second)
	if err != nil {
		log.Errorf("reconnect peer failed , err: %s", err)
		return err
	}
	log.Debugf("reconnect peer success: %s", walletAddr)
	return nil
}

// SendOnce send msg to peer asynchronous
// peer can be addr(string) or client(*network.peerClient)
func (this *Network) SendOnce(msg proto.Message, walletAddr string) error {
	if strings.Contains(walletAddr, "tcp") {
		log.Errorf("wallet addr is wrong %s", walletAddr)
		panic("wallet addr is wrong ")
		os.Exit(1)
	}
	if err := this.HealthCheckPeer(this.walletAddrFromPeerId(this.GetProxyServer().PeerID)); err != nil {
		log.Errorf("Send HealthCheckPeer walletAddrFromPeerId error: %s", err.Error())
		return err
	}
	if err := this.HealthCheckPeer(walletAddr); err != nil {
		log.Errorf("Send HealthCheckPeer HealthCheckPeer error: %s", err.Error())
		return err
	}
	pr := this.GetPeerFromWalletAddr(walletAddr)
	if pr == nil {
		return fmt.Errorf("peer not found %s", walletAddr)
	}
	peerId := pr.GetPeerId()
	state, _ := this.GetConnStateByWallet(peerId)
	if state != network.PEER_REACHABLE {
		return fmt.Errorf("can not send to inactive peer %s", peerId)
	}
	signed, err := this.P2p.PrepareMessage(context.Background(), msg)
	if err != nil {
		return fmt.Errorf("failed to sign message")
	}
	err = this.P2p.Write(peerId, signed)
	log.Debugf("write msg done sender:%s, to %s, nonce: %d", signed.GetSender().Address, peerId, signed.GetMessageNonce())
	if err != nil {
		return fmt.Errorf("failed to send message to %s", peerId)
	}
	return nil
}

// Send send msg to peer asynchronous
// peer can be addr(string) or client(*network.peerClient)
func (this *Network) Send(msg proto.Message, sessionId, msgId, walletAddr string, sendTimeout time.Duration) error {
	if strings.Contains(walletAddr, "tcp") {
		log.Errorf("wallet addr is wrong %s", walletAddr)
		panic("wallet addr is wrong ")
		os.Exit(1)
	}
	if err := this.HealthCheckPeer(this.walletAddrFromPeerId(this.GetProxyServer().PeerID)); err != nil {
		return err
	}
	log.Debugf("before send, health check it")
	if err := this.HealthCheckPeer(walletAddr); err != nil {
		return err
	}
	pr := this.GetPeerFromWalletAddr(walletAddr)
	if pr == nil {
		return fmt.Errorf("peer not found %s", walletAddr)
	}
	peerId := pr.GetPeerId()
	state, _ := this.GetConnStateByWallet(peerId)
	if state != network.PEER_REACHABLE {
		if err := this.HealthCheckPeer(walletAddr); err != nil {
			return fmt.Errorf("can not send to inactive peer %s", peerId)
		}
	}
	this.stopKeepAlive()
	log.Debugf("send msg %s to %s", msgId, peerId)
	var err error
	if sendTimeout > 0 {
		err = pr.StreamSend(sessionId, msgId, msg, sendTimeout)
	} else {
		err = pr.Send(msgId, msg)
	}
	log.Debugf("send msg %s to %s done", msgId, peerId)
	this.restartKeepAlive()
	return err
}

// Request. send msg to peer and wait for response synchronously with timeout
func (this *Network) SendAndWaitReply(msg proto.Message, sessionId, msgId, walletAddr string, sendTimeout time.Duration) (
	proto.Message, error) {
	if strings.Contains(walletAddr, "tcp") {
		log.Errorf("wallet addr is wrong %s", walletAddr)
		panic("wallet addr is wrong ")
		os.Exit(1)
	}
	if this == nil {
		return nil, errors.New("no network")
	}
	if err := this.HealthCheckPeer(this.walletAddrFromPeerId(this.GetProxyServer().PeerID)); err != nil {
		return nil, err
	}
	if err := this.HealthCheckPeer(walletAddr); err != nil {
		return nil, err
	}
	pr := this.GetPeerFromWalletAddr(walletAddr)
	if pr == nil {
		return nil, fmt.Errorf("peer not found %s", walletAddr)
	}
	peerId := pr.GetPeerId()
	state, _ := this.GetConnStateByWallet(peerId)
	if state != network.PEER_REACHABLE {
		if err := this.HealthCheckPeer(walletAddr); err != nil {
			return nil, fmt.Errorf("can not send to inactive peer %s", peerId)
		}
	}
	this.stopKeepAlive()
	log.Debugf("send msg %s to %s", msgId, peerId)
	var err error
	var resp proto.Message
	if sendTimeout > 0 {
		resp, err = pr.StreamSendAndWaitReply(sessionId, msgId, msg, sendTimeout)
	} else {
		resp, err = pr.SendAndWaitReply(msgId, msg)
	}
	log.Debugf("send and wait reply done  %s", err)
	this.restartKeepAlive()
	return resp, err
}

func (this *Network) AppendAddrToHealthCheck(walletAddr string) {
	if strings.Contains(walletAddr, "tcp") {
		log.Errorf("wallet addr is wrong %s", walletAddr)
		panic("wallet addr is wrong ")
		os.Exit(1)
	}
	pr := this.GetPeerFromWalletAddr(walletAddr)
	if pr == nil {
		return
	}
	peerId := pr.GetPeerId()
	this.peerForHealthCheck.Store(walletAddr, peerId)
}

func (this *Network) RemoveAddrFromHealthCheck(walletAddr string) {
	if strings.Contains(walletAddr, "tcp") {
		log.Errorf("wallet addr is wrong %s", walletAddr)
		panic("wallet addr is wrong ")
		os.Exit(1)
	}
	this.peerForHealthCheck.Delete(walletAddr)
}

// ClosePeerSession
func (this *Network) ClosePeerSession(walletAddr, sessionId string) error {
	if strings.Contains(walletAddr, "tcp") {
		log.Errorf("wallet addr is wrong %s", walletAddr)
		panic("wallet addr is wrong ")
		os.Exit(1)
	}
	p, ok := this.peers.Load(walletAddr)
	if !ok {
		return fmt.Errorf("peer %s not found", walletAddr)
	}
	pr := p.(*peer.Peer)
	return pr.CloseSession(sessionId)
}

// GetPeerSendSpeed. return send speed for peer
func (this *Network) GetPeerSessionSpeed(walletAddr, sessionId string) (uint64, uint64, error) {
	if strings.Contains(walletAddr, "tcp") {
		log.Errorf("wallet addr is wrong %s", walletAddr)
		panic("wallet addr is wrong ")
		os.Exit(1)
	}
	p, ok := this.peers.Load(walletAddr)
	if !ok {
		return 0, 0, fmt.Errorf("peer %s not found", walletAddr)
	}
	pr := p.(*peer.Peer)
	return pr.GetSessionSpeed(sessionId)
}

//P2P network msg receive. transfer to actor_channel
func (this *Network) Receive(ctx *network.ComponentContext, message proto.Message, hostAddr, peerId string) error {
	// TODO check message is nil
	walletAddr := this.walletAddrFromPeerId(peerId)
	switch message.(type) {
	case *messages.Processed:
		act.OnBusinessMessage(message, walletAddr)
	case *messages.Delivered:
		act.OnBusinessMessage(message, walletAddr)
	case *messages.SecretRequest:
		act.OnBusinessMessage(message, walletAddr)
	case *messages.RevealSecret:
		act.OnBusinessMessage(message, walletAddr)
	case *messages.BalanceProof:
		act.OnBusinessMessage(message, walletAddr)
	case *messages.DirectTransfer:
		act.OnBusinessMessage(message, walletAddr)
	case *messages.LockedTransfer:
		act.OnBusinessMessage(message, walletAddr)
	case *messages.RefundTransfer:
		act.OnBusinessMessage(message, walletAddr)
	case *messages.LockExpired:
		act.OnBusinessMessage(message, walletAddr)
	case *messages.WithdrawRequest:
		act.OnBusinessMessage(message, walletAddr)
	case *messages.Withdraw:
		act.OnBusinessMessage(message, walletAddr)
	case *messages.CooperativeSettleRequest:
		act.OnBusinessMessage(message, walletAddr)
	case *messages.CooperativeSettle:
		act.OnBusinessMessage(message, walletAddr)
	default:
		if len(message.String()) == 0 {
			log.Warnf("Receive keepalive request/response msg from %s", walletAddr)
			return nil
		}
		log.Errorf("[P2pNetwork Receive] unknown message type:%s type %T", message.String(), message)
	}

	return nil
}

func (this *Network) GetClientTime(walletAddr string) (uint64, error) {
	if strings.Contains(walletAddr, "tcp") {
		log.Errorf("wallet addr is wrong %s", walletAddr)
		panic("wallet addr is wrong ")
		os.Exit(1)
	}
	p, ok := this.peers.Load(walletAddr)
	if !ok {
		return 0, fmt.Errorf("[network GetClientTime] client is nil: %s", walletAddr)
	}
	t := p.(*peer.Peer).ActiveTime()
	return uint64(t.Unix()), nil
}

func (this *Network) IsPeerNetQualityBad(walletAddr string) bool {
	if strings.Contains(walletAddr, "tcp") {
		log.Errorf("wallet addr is wrong %s", walletAddr)
		panic("wallet addr is wrong ")
		os.Exit(1)
	}
	totalFailed := &peer.FailedCount{}
	peerCnt := 0
	var peerFailed *peer.FailedCount
	this.peers.Range(func(key, value interface{}) bool {
		peerCnt++
		peerWalletAddr, _ := key.(string)
		pr, _ := value.(*peer.Peer)
		cnt := pr.GetFailedCnt()
		totalFailed.Dial += cnt.Dial
		totalFailed.Send += cnt.Send
		totalFailed.Recv += cnt.Recv
		totalFailed.Disconnect += cnt.Disconnect
		if peerWalletAddr == walletAddr {
			peerFailed = cnt
		}
		return false
	})
	if peerFailed == nil {
		return false
	}
	log.Debugf("peer failed %v, totalFailed %v, peer cnt %v", peerFailed, totalFailed, peerCnt)
	if (peerFailed.Dial > 0 && peerFailed.Dial >= totalFailed.Dial/peerCnt) ||
		(peerFailed.Send > 0 && peerFailed.Send >= totalFailed.Send/peerCnt) ||
		(peerFailed.Disconnect > 0 && peerFailed.Disconnect >= totalFailed.Disconnect/peerCnt) ||
		(peerFailed.Recv > 0 && peerFailed.Recv >= totalFailed.Recv/peerCnt) {
		return true
	}
	return false
}

func (this *Network) reconnect(walletAddr string) error {
	p, ok := this.peers.Load(walletAddr)
	var pr *peer.Peer
	if !ok {
		return fmt.Errorf("peer not found %s", walletAddr)
	}
	pr = p.(*peer.Peer)
	if len(walletAddr) > 0 && pr.State() == peer.ConnectStateConnecting {
		if err := this.WaitForConnected(walletAddr, time.Duration(15)*time.Second); err != nil {
			pr.SetState(peer.ConnectStateFailed)
			return err
		}
		pr.SetState(peer.ConnectStateConnected)
		return nil
	}
	pr.SetState(peer.ConnectStateConnecting)
	if len(walletAddr) > 0 {
		state, err := this.GetConnStateByWallet(walletAddr)
		if state == network.PEER_REACHABLE && err == nil {
			pr.SetState(peer.ConnectStateConnected)
			return nil
		}
	}
	err, peerId := this.P2p.ReconnectPeer(pr.GetHostAddr())
	if err != nil {
		pr.SetState(peer.ConnectStateFailed)
	} else {
		pr.SetState(peer.ConnectStateConnected)
	}
	pr.SetPeerId(peerId)
	this.peers.Store(walletAddr, pr)
	return err
}

//
//func (this *Network) addProxyComponents(builder *network.Builder) {
//	servers := strings.Split(config.Parameters.BaseConfig.NATProxyServerAddrs, ",")
//	hasAdd := make(map[string]struct{})
//	for _, proxyAddr := range servers {
//		if len(proxyAddr) == 0 {
//			continue
//		}
//		protocol := getProtocolFromAddr(proxyAddr)
//		_, ok := hasAdd[protocol]
//		if ok {
//			continue
//		}
//		switch protocol {
//		case "udp":
//			builder.AddComponent(new(proxy.UDPProxyComponent))
//		case "kcp":
//			builder.AddComponent(new(proxy.KCPProxyComponent))
//		case "quic":
//			builder.AddComponent(new(proxy.QuicProxyComponent))
//		case "tcp":
//			builder.AddComponent(new(proxy.TcpProxyComponent))
//		}
//		hasAdd[protocol] = struct{}{}
//	}
//}

func (this *Network) healthCheckService() {
	ti := time.NewTicker(time.Duration(5) * time.Second)
	startedAt := time.Now().Unix()
	for {
		select {
		case <-ti.C:
			shouldLog := ((time.Now().Unix()-startedAt)%60 == 0)
			//this.HealthCheckPeer(this.walletAddrFromPeerId(this.GetProxyServer().PeerID))
			this.peerForHealthCheck.Range(func(key, value interface{}) bool {
				addr, _ := key.(string)
				if len(addr) == 0 {
					return true
				}
				if shouldLog {
					addrState, err := this.GetConnStateByWallet(addr)
					if err != nil {
						log.Errorf("publicAddr: %s, addr %s state: %d, err: %s",
							this.PublicAddr(), addr, addrState, err)
					} else {
						log.Debugf("publicAddr: %s, addr %s state: %d", this.PublicAddr(), addr, addrState)
					}
				}
				this.HealthCheckPeer(addr)
				return true
			})
		case <-this.kill:
			log.Debugf("stop health check proxy service when receive kill")
			ti.Stop()
			return
		}
	}
}

func (this *Network) getKeepAliveComponent() *keepalive.Component {
	for _, info := range this.P2p.Components.GetInstallComponents() {
		keepaliveCom, ok := info.Component.(*keepalive.Component)
		if !ok {
			continue
		}
		return keepaliveCom
	}
	return nil
}

func (this *Network) updatePeerTime(walletAddr string) {
	pr := this.GetPeerFromWalletAddr(walletAddr)
	if pr == nil {
		return
	}
	peerId := pr.GetPeerId()
	client := this.P2p.GetPeerClient(peerId)
	if client == nil {
		return
	}
	client.Time = time.Now()
}

func (this *Network) stopKeepAlive() {
	this.lock.Lock()
	defer this.lock.Unlock()
	var ka *keepalive.Component
	var ok bool
	for _, info := range this.P2p.Components.GetInstallComponents() {
		ka, ok = info.Component.(*keepalive.Component)
		if ok {
			break
		}
	}
	if ka == nil {
		return
	}
	// stop keepalive for temporary
	ka.Cleanup(this.P2p)
	deleted := this.P2p.Components.Delete(ka)
	log.Debugf("stop keep alive %t", deleted)
}

func (this *Network) restartKeepAlive() {
	this.lock.Lock()
	defer this.lock.Unlock()
	var ka *keepalive.Component
	var ok bool
	for _, info := range this.P2p.Components.GetInstallComponents() {
		ka, ok = info.Component.(*keepalive.Component)
		if ok {
			break
		}
	}
	if ka != nil {
		return
	}
	options := []keepalive.ComponentOption{
		keepalive.WithKeepaliveInterval(keepalive.DefaultKeepaliveInterval),
		keepalive.WithKeepaliveTimeout(keepalive.DefaultKeepaliveTimeout),
	}
	ka = keepalive.New(options...)
	err := this.builder.AddComponent(ka)
	if err != nil {
		return
	}
	ka.Startup(this.P2p)
}

// startProxy. start proxy service
//func (this *Network) startProxy(builder *network.Builder) error {
//	var err error
//	log.Debugf("NATProxyServerAddrs :%v", this.proxyAddrs)
//	for _, proxyAddr := range this.proxyAddrs {
//		if len(proxyAddr) == 0 {
//			continue
//		}
//		this.P2p.EnableProxyMode(true)
//		this.P2p.SetProxyServer([]network.ProxyServer{
//			network.ProxyServer{
//				IP: proxyAddr,
//			},
//		})
//		protocol := getProtocolFromAddr(proxyAddr)
//		log.Debugf("start proxy will blocking...%s %s, networkId: %d",
//			protocol, proxyAddr, config.Parameters.BaseConfig.NetworkId)
//		done := make(chan struct{}, 1)
//		go func() {
//			switch protocol {
//			case "udp":
//				this.P2p.BlockUntilUDPProxyFinish()
//			case "kcp":
//				this.P2p.BlockUntilKCPProxyFinish()
//			case "quic":
//				this.P2p.BlockUntilQuicProxyFinish()
//			case "tcp":
//				this.P2p.BlockUntilTcpProxyFinish()
//			}
//			done <- struct{}{}
//		}()
//		select {
//		case <-done:
//			proxyHostAddr, proxyPeerId := this.P2p.GetWorkingProxyServer()
//			pr := peer.New(proxyHostAddr)
//			pr.SetPeerId(proxyPeerId)
//			this.peers.Store(proxyPeerId, pr)
//			log.Debugf("start proxy finish, publicAddr: %s, proxy peer id %s", this.P2p.ID.Address, proxyPeerId)
//			return nil
//		case <-time.After(time.Duration(20) * time.Second):
//			err = fmt.Errorf("proxy: %s timeout", proxyAddr)
//			log.Debugf("start proxy err :%s", err)
//			break
//		}
//	}
//	return err
//}

func getProtocolFromAddr(addr string) string {
	idx := strings.Index(addr, "://")
	if idx == -1 {
		return "tcp"
	}
	return addr[:idx]
}
