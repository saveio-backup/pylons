package transport

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/saveio/carrier/network/components/keepalive"
	"github.com/saveio/pylons/actor/client"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/pylons/network/transport/messages"
	"github.com/saveio/pylons/transfer"
	chainComm "github.com/saveio/themis/common"
	"github.com/saveio/themis/common/log"
)

type ChannelServiceInterface interface {
	OnMessage(proto.Message, string)
	Sign(message interface{}) error
	HandleStateChange(stateChange transfer.StateChange) []transfer.Event
	StateFromChannel() *transfer.ChainState
}

type Transport struct {
	NodeIpPortToAddress *sync.Map
	NodeAddressToIpPort *sync.Map

	messageQueues       *sync.Map
	addressQueueMap     *sync.Map
	kill                chan struct{}
	getHostAddrCallback func(address common.Address) (string, error)
	ChannelService      ChannelServiceInterface
}

type QueueItem struct {
	message   proto.Message
	messageId *messages.MessageID
}

func NewTransport(channelService ChannelServiceInterface) *Transport {
	return &Transport{
		kill:                make(chan struct{}),
		messageQueues:       new(sync.Map),
		addressQueueMap:     new(sync.Map),
		NodeIpPortToAddress: new(sync.Map),
		NodeAddressToIpPort: new(sync.Map),
		ChannelService:      channelService,
	}
}

// messages first be queued, only can be send when Delivered for previous msssage is received
func (this *Transport) SendAsync(queueId *transfer.QueueIdentifier, msg proto.Message) error {
	var msgID *messages.MessageID
	rec := chainComm.Address(queueId.Recipient)
	log.Debugf("[SendAsync] %v, TO: %v.", reflect.TypeOf(msg).String(), rec.ToBase58())
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
	case *messages.RefundTransfer:
		msgID = (msg.(*messages.RefundTransfer)).Refund.BaseMessage.MessageIdentifier
	case *messages.Secret:
		msgID = (msg.(*messages.Secret)).MessageIdentifier
	case *messages.LockExpired:
		msgID = (msg.(*messages.LockExpired)).MessageIdentifier
	case *messages.WithdrawRequest:
		msgID = (msg.(*messages.WithdrawRequest)).MessageIdentifier
	case *messages.Withdraw:
		msgID = (msg.(*messages.Withdraw)).MessageIdentifier
	case *messages.CooperativeSettleRequest:
		msgID = (msg.(*messages.CooperativeSettleRequest)).MessageIdentifier
	case *messages.CooperativeSettle:
		msgID = (msg.(*messages.CooperativeSettle)).MessageIdentifier
	default:
		log.Error("[SendAsync] Unknown message type to send async: ", reflect.TypeOf(msg).String())
		return fmt.Errorf("Unknown message type to send async ")
	}

	//log.Infof("[SendAsync] %v, msgId: %d, TO: %v.", reflect.TypeOf(msg).String(), msgID, rec.ToBase58())
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

	// queueid cannot be pointer type otherwise it might be updated outside QueueSend
	go this.QueueSend(q, *queueId)

	return q
}

func (this *Transport) QueueSend(queue *Queue, queueId transfer.QueueIdentifier) {
	var interval time.Duration = 3
	var retryTimes = 0

	t := time.NewTimer(interval * time.Second)

	for {
		select {
		case <-queue.DataCh:
			log.Debugf("[QueueSend] <-queue.DataCh Time: %s queue: %p\n", time.Now().String(), queue)
			t.Reset(interval * time.Second)
			this.PeekAndSend(queue, &queueId)
		// handle timeout retry
		case <-t.C:
			log.Debugf("[QueueSend]  <-t.C Time: %s queue: %p\n", time.Now().String(), queue)
			if queue.Len() == 0 {
				continue
			}

			item, _ := queue.Peek()
			msg := item.(*QueueItem).message
			log.Warnf("Timeout retry for msg = %+v\n", msg)

			t.Reset((time.Duration)(retryTimes+1) * interval * time.Second)
			err := this.PeekAndSend(queue, &queueId)
			if err != nil {
				log.Errorf("send message to %s failed: %s", common.ToBase58(queueId.Recipient), err)
				// dont stop the timer, otherwise it will block trying to resend message
				//t.Stop()
				//break
			}
			retryTimes++
		case msgId := <-queue.DeliverChan:
			log.Debugf("[DeliverChan] Time: %s msgId := <-queue.DeliverChan queue: %p msgId = %+v queue.length: %d\n",
				time.Now().String(), queue, msgId.MessageId, queue.Len())
			data, _ := queue.Peek()
			if data == nil {
				log.Debug("[DeliverChan] msgId := <-queue.DeliverChan data == nil")
				log.Error("msgId := <-queue.DeliverChan data == nil")
				continue
			}
			item := data.(*QueueItem)
			log.Debugf("[DeliverChan] msgId := <-queue.DeliverChan: %s item = %+v\n",
				reflect.TypeOf(item.message).String(), item.messageId)
			if msgId.MessageId == item.messageId.MessageId {
				this.addressQueueMap.Delete(msgId.MessageId)

				queue.Pop()
				t.Stop()
				if queue.Len() != 0 {
					log.Debug("msgId.MessageId == item.messageId.MessageId queue.Len() != 0")
					t.Reset(interval * time.Second)
					retryTimes = 0
					this.PeekAndSend(queue, &queueId)
				}
			} else {
				log.Debug("[DeliverChan] msgId.MessageId != item.messageId.MessageId queue.Len: ", queue.Len())
				log.Warnf("[DeliverChan] msgId.MessageId: %d != item.messageId.MessageId: %d", msgId.MessageId,
					item.messageId.MessageId)
			}
		case <-this.kill:
			log.Info("[QueueSend] msgId := <-this.kill")
			t.Stop()
			return
		}
	}
}

func (this *Transport) Stop() {
	close(this.kill)
	log.Debug("transport stopped")
}

func (this *Transport) PeekAndSend(queue *Queue, queueId *transfer.QueueIdentifier) error {
	item, ok := queue.Peek()
	if !ok {
		return fmt.Errorf("Error peeking from queue. ")
	}

	msg := item.(*QueueItem).message
	log.Debugf("send msg msg = %+v\n", msg)
	address, err := this.GetHostAddr(queueId.Recipient)
	if address == "" || err != nil {
		log.Errorf("[PeekAndSend] GetHostAddr address is nil for %s", common.ToBase58(queueId.Recipient))
		return errors.New("no valid address to send message")
	}

	state := this.GetNodeNetworkState(address)
	if state != transfer.NetworkReachable {
		log.Errorf("[PeekAndSend] state != NetworkReachable %s", address)
		//log.Warnf("[PeekAndSend] state != NetworkReachable reconnect %s", address)
		//if err = client.P2pConnect(address); err != nil {
		//	log.Errorf("[PeekAndSend] state != NetworkReachable connect error: %s", err.Error())
		//}
	}

	msgId := common.MessageID(item.(*QueueItem).messageId.MessageId)
	log.Debugf("[PeekAndSend] address: %s msgId: %v, queue: %p\n", address, msgId, queue)

	this.addressQueueMap.LoadOrStore(msgId, queue)
	if err = client.P2pSend(address, msg); err != nil {
		log.Error("[PeekAndSend] send error: ", err.Error())
		return err
	}

	return nil
}

func (this *Transport) Send(address common.Address, msg proto.Message) error {
	nodeAddress, err := this.GetHostAddr(address)
	if nodeAddress == "" || err != nil {
		log.Error("[Send] GetHostAddr address is nil for %s", common.ToBase58(address))
		return errors.New("no valid address to send message")
	}

	if err = client.P2pSend(nodeAddress, msg); err != nil {
		log.Error("[PeekAndSend] send error: ", err.Error())
		return err
	}
	return nil
}

func (this *Transport) SetGetHostAddrCallback(getHostAddrCallback func(address common.Address) (string, error)) {
	this.getHostAddrCallback = getHostAddrCallback
}

func (this *Transport) SetHostAddr(address common.Address, hostAddr string) {
	this.NodeAddressToIpPort.Store(address, hostAddr)
	this.NodeIpPortToAddress.Store(hostAddr, address)
}

func (this *Transport) GetHostAddrByCallBack(walletAddr common.Address) (string, error) {
	var err error
	var nodeNetAddr string

	if this.getHostAddrCallback != nil {
		nodeNetAddr, err = this.getHostAddrCallback(walletAddr)
		if err == nil {
			if this.GetNodeNetworkState(nodeNetAddr) == transfer.NetworkReachable {
				this.NodeAddressToIpPort.Store(walletAddr, nodeNetAddr)
				this.NodeIpPortToAddress.Store(nodeNetAddr, walletAddr)
				return nodeNetAddr, nil
			} else {
				err = fmt.Errorf("[GetHostAddrByCallBack] %s is not reachable", nodeNetAddr)
			}
		}
	} else {
		err = fmt.Errorf("[GetHostAddrByCallBack] error: getHostAddrCallback is not set")
	}
	log.Errorf("[GetHostAddrByCallBack] error: %s", err.Error())
	return "", err
}

func (this *Transport) GetHostAddr(walletAddr common.Address) (string, error) {
	var err error
	var nodeNetAddr string

	if v, ok := this.NodeAddressToIpPort.Load(walletAddr); ok {
		nodeNetAddr = v.(string)
		if this.GetNodeNetworkState(nodeNetAddr) == transfer.NetworkReachable {
			return nodeNetAddr, nil
		} else {
			err = fmt.Errorf("[GetHostAddr] %s is not reachable", nodeNetAddr)
		}
	}
	log.Warnf("[GetHostAddrFromLocal] error: %s. Try GetHostAddrByCallBack", err.Error())
	nodeNetAddr, err = this.GetHostAddrByCallBack(walletAddr)
	return nodeNetAddr, err
}

func (this *Transport) StartHealthCheck(address common.Address) {
	log.Infof("[StartHealthCheck] address: %s", common.ToBase58(address))
	nodeAddress, err := this.GetHostAddr(address)
	if nodeAddress == "" || err != nil {
		log.Error("node address invalid, can`t check health")
		return
	}
	client.P2pConnect(nodeAddress)
}

func (this *Transport) GetNodeNetworkState(nodeNetAddress string) string {
	nodeState := client.GetNodeNetworkState(nodeNetAddress)
	if nodeState != int(keepalive.PEER_REACHABLE) {
		return transfer.NetworkUnreachable
	} else {
		return transfer.NetworkReachable
	}

}

func (this *Transport) Receive(message proto.Message, from string) {
	log.Info("[NetComponent] Receive: ", reflect.TypeOf(message).String(), " From: ", from)

	switch message.(type) {
	case *messages.Delivered:
		go this.ReceiveDelivered(message, from)
	default:
		go this.ReceiveMessage(message, from)
	}
}

func (this *Transport) ReceiveMessage(message proto.Message, fromNetAddr string) {
	log.Debugf("[ReceiveMessage] %v from: %v", reflect.TypeOf(message).String(), fromNetAddr)
	if this.ChannelService != nil {
		this.ChannelService.OnMessage(message, fromNetAddr)
	}
	var senderWallerAddr common.Address
	var msgID *messages.MessageID

	switch message.(type) {
	case *messages.DirectTransfer:
		msg := message.(*messages.DirectTransfer)
		senderWallerAddr = messages.ConvertAddress(msg.EnvelopeMessage.Signature.Sender)
		msgID = msg.MessageIdentifier
	case *messages.Processed:
		msg := message.(*messages.Processed)
		senderWallerAddr = messages.ConvertAddress(msg.Signature.Sender)
		msgID = msg.MessageIdentifier
	case *messages.LockedTransfer:
		msg := message.(*messages.LockedTransfer)
		senderWallerAddr = messages.ConvertAddress(msg.BaseMessage.EnvelopeMessage.Signature.Sender)
		msgID = msg.BaseMessage.MessageIdentifier
	case *messages.LockExpired:
		msg := message.(*messages.LockExpired)
		senderWallerAddr = messages.ConvertAddress(msg.EnvelopeMessage.Signature.Sender)
		msgID = msg.MessageIdentifier
	case *messages.RefundTransfer:
		msg := message.(*messages.RefundTransfer)
		senderWallerAddr = messages.ConvertAddress(msg.Refund.BaseMessage.EnvelopeMessage.Signature.Sender)
		msgID = msg.Refund.BaseMessage.MessageIdentifier
	case *messages.SecretRequest:
		msg := message.(*messages.SecretRequest)
		senderWallerAddr = messages.ConvertAddress(msg.Signature.Sender)
		msgID = msg.MessageIdentifier
	case *messages.RevealSecret:
		msg := message.(*messages.RevealSecret)
		senderWallerAddr = messages.ConvertAddress(msg.Signature.Sender)
		msgID = msg.MessageIdentifier
	case *messages.Secret:
		msg := message.(*messages.Secret)
		senderWallerAddr = messages.ConvertAddress(msg.EnvelopeMessage.Signature.Sender)
		msgID = msg.MessageIdentifier
	case *messages.WithdrawRequest:
		msg := message.(*messages.WithdrawRequest)
		senderWallerAddr = messages.ConvertAddress(msg.Participant)
		msgID = msg.MessageIdentifier
	case *messages.Withdraw:
		msg := message.(*messages.Withdraw)
		senderWallerAddr = messages.ConvertAddress(msg.PartnerSignature.Sender)
		msgID = msg.MessageIdentifier
	case *messages.CooperativeSettleRequest:
		msg := message.(*messages.CooperativeSettleRequest)
		senderWallerAddr = messages.ConvertAddress(msg.Participant1Signature.Sender)
		msgID = msg.MessageIdentifier
	case *messages.CooperativeSettle:
		msg := message.(*messages.CooperativeSettle)
		senderWallerAddr = messages.ConvertAddress(msg.Participant2Signature.Sender)
		msgID = msg.MessageIdentifier
	default:
		log.Warn("[ReceiveMessage] unknown Msg type: ", reflect.TypeOf(message).String())
		return
	}

	log.Debugf("[ReceiveMessage] %s msgId: %d fromNetAddr: %s fromWalletAddr: %s", reflect.TypeOf(message).String(),
		msgID.MessageId, fromNetAddr, common.ToBase58(senderWallerAddr))
	deliveredMessage := &messages.Delivered{
		DeliveredMessageIdentifier: msgID,
	}

	//var nodeNetAddress string
	err := this.ChannelService.Sign(deliveredMessage)
	if err == nil {
		if senderWallerAddr != common.EmptyAddress {
			this.SetHostAddr(senderWallerAddr, fromNetAddr)
		}
		log.Debugf("SendDeliveredMessage (%v) Time: %s DeliveredMessageIdentifier: %v deliveredMessage from: %v",
			reflect.TypeOf(message).String(), time.Now().String(), deliveredMessage.DeliveredMessageIdentifier.MessageId,
			fromNetAddr)

		state := this.GetNodeNetworkState(fromNetAddr)
		if state != transfer.NetworkReachable {
			log.Errorf("[PeekAndSend] state != NetworkReachable %s", fromNetAddr)
			//log.Warn("[PeekAndSend] state != NetworkReachable reconnect %s", nodeNetAddress)
			//if err = client.P2pConnect(nodeNetAddress); err != nil {
			//	log.Errorf("[PeekAndSend] state != NetworkReachable connect error: %s", err.Error())
			//}
		}

		if err = client.P2pSend(fromNetAddr, deliveredMessage); err != nil {
			log.Errorf("SendDeliveredMessage (%v) Time: %s DeliveredMessageIdentifier: %v deliveredMessage from: %v error: %s",
				reflect.TypeOf(message).String(), time.Now().String(), deliveredMessage.DeliveredMessageIdentifier.MessageId,
				fromNetAddr, err.Error())
		}
	} else {
		log.Errorf("SendDeliveredMessage (%v) deliveredMessage Sign error: ", err.Error(),
			reflect.TypeOf(message).String(), fromNetAddr)
	}
}

func (this *Transport) ReceiveDelivered(message proto.Message, from string) {
	if this.ChannelService != nil {
		this.ChannelService.OnMessage(message, from)
	}
	//f := func(key, value interface{}) bool {
	//	fmt.Printf("[ReceiveDelivered] addressQueueMap Content \n", )
	//	fmt.Printf("k Type: %s k %v \n", reflect.TypeOf(key).String(), key)
	//	fmt.Printf("v Type: %s v %v \n", reflect.TypeOf(value).String(), value)
	//	return true
	//}
	msg := message.(*messages.Delivered)
	msgId := common.MessageID(msg.DeliveredMessageIdentifier.MessageId)
	queue, ok := this.addressQueueMap.Load(msgId)
	if !ok {
		log.Debugf("[ReceiveDelivered] from: %s Time: %s msgId: %v\n", from, time.Now().String(), msgId)
		log.Error("[ReceiveDelivered] msg.DeliveredMessageIdentifier is not in addressQueueMap")
		return
	}
	log.Debugf("[ReceiveDelivered] from: %s Time: %s msgId: %v, queue: %p\n", from, time.Now().String(), msgId, queue)
	queue.(*Queue).DeliverChan <- msg.DeliveredMessageIdentifier
}
