package network

import (
	"github.com/saveio/carrier/network"
	"github.com/saveio/edge/p2p/peer"
	"github.com/saveio/themis/common/log"
)

type PeerComponent struct {
	Net *Network
}

func (this *PeerComponent) Startup(net *network.Network) {
}

func (this *PeerComponent) Cleanup(net *network.Network) {
}

func (this *PeerComponent) Receive(ctx *network.ComponentContext) error {
	return nil
}

func (this *PeerComponent) PeerConnect(client *network.PeerClient) {
	if client == nil || len(client.Address) == 0 {
		log.Warnf("peer has connected, but client is nil", client)
		return
	}
	hostAddr := client.Address
	if this.Net.IsProxyAddr(hostAddr) {
		return
	}
	peerId := client.ClientID()
	walletAddr := this.Net.walletAddrFromPeerId(peerId)
	p, ok := this.Net.peers.LoadOrStore(walletAddr, peer.New(hostAddr))
	pr, ok := p.(*peer.Peer)
	if !ok {
		log.Errorf("convert peer to peer.Peer failed")
		return
	}
	log.Infof("peer %s has connected, peer id is %s", hostAddr, peerId)
	pr.SetClient(client)
	pr.SetPeerId(peerId)
}

func (this *PeerComponent) PeerDisconnect(client *network.PeerClient) {
	if client == nil || len(client.Address) == 0 {
		log.Warnf("peer has disconnected, but its address is clean")
		return
	}
	peerId := client.ClientID()
	walletAddr := this.Net.walletAddrFromPeerId(peerId)
	log.Debugf("peer has disconnected, health check peer %s", walletAddr)
	this.Net.HealthCheckPeer(walletAddr)
}
