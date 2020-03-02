package server

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/ontio/ontology-eventbus/actor"
	p2pNet "github.com/saveio/carrier/network"
	chact "github.com/saveio/pylons/actor/client"
	"github.com/saveio/pylons/test/p2p/network"
	"github.com/saveio/themis/common/log"
	"time"
)

type MessageHandler func(msgData interface{}, pid *actor.PID)

type P2PActor struct {
	channelNet          *network.Network
	dspNet              *network.Network
	props               *actor.Props
	msgHandlers         map[string]MessageHandler
	localPID            *actor.PID
	getHostAddrCallBack func(string) (string, error)
}

func NewP2PActor() (*P2PActor, error) {
	var err error
	p2pActor := &P2PActor{
		msgHandlers: make(map[string]MessageHandler),
	}
	p2pActor.localPID, err = p2pActor.Start()
	if err != nil {
		return nil, err
	}
	return p2pActor, nil
}

func (this *P2PActor) SetHostAddrCallback(cb func(address string) (string, error)) {
	this.getHostAddrCallBack = cb
}

func (this *P2PActor) SetChannelNetwork(net *network.Network) {
	this.channelNet = net
}

func (this *P2PActor) SetDspNetwork(net *network.Network) {
	this.dspNet = net
}

func (this *P2PActor) Start() (*actor.PID, error) {
	this.props = actor.FromProducer(func() actor.Actor { return this })
	localPid, err := actor.SpawnNamed(this.props, "net_server")
	if err != nil {
		return nil, fmt.Errorf("[P2PActor] start error:%v", err)
	}
	this.localPID = localPid
	return localPid, err
}

func (this *P2PActor) Stop() error {
	this.localPID.Stop()
	this.dspNet.Stop()
	this.channelNet.Stop()
	return nil
}

func (this *P2PActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Restarting:
		log.Warn("[P2PActor] actor restarting")
	case *actor.Stopping:
		log.Warn("[P2PActor] actor stopping")
	case *actor.Stopped:
		log.Warn("[P2PActor] actor stopped")
	case *actor.Started:
		log.Debug("[P2PActor] actor started")
	case *actor.Restart:
		log.Warn("[P2PActor] actor restart")
	case *chact.ConnectReq:
		go func() {
			targetNetAddr, err := this.getHostAddrCallBack(msg.Address)
			if err != nil {
				msg.Ret.Err = err
				msg.Ret.Done <- true
			}

			err = this.channelNet.Connect(targetNetAddr)
			if err != nil {
				msg.Ret.Err = err
				msg.Ret.Done <- true
			}
			err = this.channelNet.WaitForConnected(msg.Address, 15*time.Second)
			if err != nil {
				msg.Ret.Err = err
				msg.Ret.Done <- true
			}
			msg.Ret.Done <- true
		}()
	case *chact.GetNodeNetworkStateReq:
		go func() {
			state, err := this.channelNet.GetConnStateByWallet(msg.Address)
			msg.Ret.State = int(state)
			fmt.Printf("[GetConnStateByWallet] %s: %d\n", msg.Address, msg.Ret.State)
			msg.Ret.Err = err
			msg.Ret.Done <- true
		}()
	case *chact.CloseReq:
		go func() {
			//todo: Fix it
			//msg.Ret.Err = this.channelNet.Close(msg.Address)
			msg.Ret.Done <- true
		}()
	case *chact.SendReq:
		go func() {
			msg.Ret.Err = this.channelNet.SendOnce(msg.Data, msg.Address)
			msg.Ret.Done <- true
		}()
	default:
		log.Error("[P2PActor] receive unknown message type!")
	}
}

func (this *P2PActor) Broadcast(message proto.Message) {
	ctx := p2pNet.WithSignMessage(context.Background(), true)
	this.channelNet.P2p.Broadcast(ctx, message)
}

func (this *P2PActor) RegMsgHandler(msgName string, handler MessageHandler) {
	this.msgHandlers[msgName] = handler
}

func (this *P2PActor) UnRegMsgHandler(msgName string, handler MessageHandler) {
	delete(this.msgHandlers, msgName)
}

func (this *P2PActor) GetLocalPID() *actor.PID {
	return this.localPID
}
