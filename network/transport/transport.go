package transport

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/saveio/carrier/network"
	"github.com/saveio/pylons/actor/client"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
	"github.com/saveio/pylons/network/transport/messages"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis/common/log"
)

const (
	QueueBusy = iota
	QueueFree
	QueueNotExist
)

const MAX_RETRY = 5

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
	targetQueueState    *sync.Map
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
		targetQueueState:    new(sync.Map),
		ChannelService:      channelService,
	}
}

// messages first be queued, only can be send when Delivered for previous msssage is received
func (this *Transport) SendAsync(queueId *transfer.QueueIdentifier, msg proto.Message) error {
	var err error
	var msgID *messages.MessageID

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
		err = fmt.Errorf("[SendAsync] Unknown message type to send async: %s", reflect.TypeOf(msg).String())
		log.Error("[SendAsync] error: ", err.Error())
		return err
	}

	log.Debugf("[SendAsync] %v, msgId: %d, TO: %v.", reflect.TypeOf(msg).String(), msgID,
		common.ToBase58(queueId.Recipient))

	q := this.GetQueue(queueId)
	if ok := q.Push(&QueueItem{message: msg, messageId: msgID}); !ok {
		err = fmt.Errorf("[SendAsync] failed to push to queue")
		log.Error("[SendAsync] error: ", err.Error())
	}
	return err
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
	this.SetTargetQueueState(queueId.Recipient, QueueFree)
	this.messageQueues.Store(*queueId, q)

	// queueId cannot be pointer type otherwise it might be updated outside QueueSend
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
			this.SetTargetQueueState(queueId.Recipient, QueueBusy)
			log.Debugf("[QueueSend]  <-t.C Time: %s queue: %p\n", time.Now().String(), queue)
			if queue.Len() == 0 {
				this.SetTargetQueueState(queueId.Recipient, QueueFree)
				continue
			}

			item, _ := queue.Peek()
			msg := item.(*QueueItem).message
			log.Warnf("Timeout retry for msg = %+v to %s\n", msg, common.ToBase58(queueId.Recipient))

			t.Reset(interval * time.Second)

			// if reach max try , only try to send when networkstate is reachable
			if retryTimes == MAX_RETRY {
				_, err := this.GetHostAddr(queueId.Recipient)
				if err != nil {
					log.Debugf("[QueueSend] reach max retry, dont send message")
					continue
				} else {
					log.Debugf("[QueueSend] network recovered, retry send message")
					retryTimes = 0
				}
			}

			err := this.PeekAndSend(queue, &queueId)
			if err != nil {
				log.Errorf("send message to %s failed: %s", common.ToBase58(queueId.Recipient), err)
				// dont stop the timer, otherwise it will block trying to resend message
				//t.Stop()
				//break
				retryTimes++
			}

		case msgId := <-queue.DeliverChan:
			log.Debugf("[DeliverChan] msgId := <-queue.DeliverChan queue: %p msgId = %d queue.length: %d\n",
				queue, msgId.MessageId, queue.Len())
			data, _ := queue.Peek()
			if data == nil {
				log.Warn("[DeliverChan] msgId := <-queue.DeliverChan data == nil")
				this.SetTargetQueueState(queueId.Recipient, QueueFree)
				continue
			}
			item := data.(*QueueItem)

			log.Debugf("[DeliverChan] msgId := <-queue.DeliverChan: %s item = %d\n",
				reflect.TypeOf(item.message).String(), item.messageId)
			if msgId.MessageId == item.messageId.MessageId {
				this.addressQueueMap.Delete(msgId.MessageId)

				queue.Pop()
				t.Stop()
				if queue.Len() != 0 {
					log.Debug("msgId.MessageId == item.messageId.MessageId queue.Len() != 0")
					t.Reset(interval * time.Second)
					this.PeekAndSend(queue, &queueId)
				} else {
					this.SetTargetQueueState(queueId.Recipient, QueueFree)
				}
				retryTimes = 0
			} else {
				log.Debug("[DeliverChan] msgId.MessageId != item.messageId.MessageId queue.Len: ", queue.Len())
				log.Warnf("[DeliverChan] MessageId not match (%d  %d)", msgId.MessageId, item.messageId.MessageId)
				this.SetTargetQueueState(queueId.Recipient, QueueBusy)
			}
		case <-this.kill:
			this.targetQueueState.Delete(queueId.Recipient)
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
	if err != nil {
		log.Errorf("[PeekAndSend] GetHostAddr %s, error: %s", common.ToBase58(queueId.Recipient), err.Error())
		return errors.New("no valid address to send message")
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
	if err != nil {
		log.Error("[Send] GetHostAddr address %s, error: %s", common.ToBase58(address), err.Error())
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

func (this *Transport) GetHostAddrByLocal(walletAddr common.Address) (string, error) {
	if v, ok := this.NodeAddressToIpPort.Load(walletAddr); ok {
		nodeNetAddr := v.(string)
		state := client.GetNodeNetworkState(nodeNetAddr)
		if state == int(network.PEER_REACHABLE) {
			return nodeNetAddr, nil
		} else {
			return "", fmt.Errorf("[GetHostAddrByLocal] %s is not reachable", common.ToBase58(walletAddr))
		}
	} else {
		return "", fmt.Errorf("[GetHostAddrByLocal] %s not found", common.ToBase58(walletAddr))
	}
}

func (this *Transport) GetHostAddrByCallBack(walletAddr common.Address) (string, error) {
	if this.getHostAddrCallback != nil {
		nodeNetAddr, err := this.getHostAddrCallback(walletAddr)
		if err == nil {
			if client.GetNodeNetworkState(nodeNetAddr) == int(network.PEER_REACHABLE) {
				this.SetHostAddr(walletAddr, nodeNetAddr)
				return nodeNetAddr, nil
			} else {
				return "", fmt.Errorf("[GetHostAddrByCallBack] %s is not reachable", common.ToBase58(walletAddr))
			}
		} else {
			return "", fmt.Errorf("[GetHostAddrByCallBack] %s error: %s", common.ToBase58(walletAddr), err.Error())
		}
	} else {
		return "", fmt.Errorf("[GetHostAddrByCallBack] error: getHostAddrCallback is not set")
	}
}

func (this *Transport) GetHostAddr(walletAddr common.Address) (string, error) {
	var err error
	var nodeNetAddr string

	nodeNetAddr, err = this.GetHostAddrByLocal(walletAddr)
	if err == nil {
		return nodeNetAddr, nil
	} else {
		log.Warnf("[GetHostAddr] GetHostAddrByLocal: %s error: %s , Try GetHostAddrByCallBack",
			common.ToBase58(walletAddr), err.Error())
	}
	nodeNetAddr, err = this.GetHostAddrByCallBack(walletAddr)
	if err != nil {
		log.Errorf("[GetHostAddr] GetHostAddrByCallBack: %s error: %s", common.ToBase58(walletAddr), err.Error())
	}
	return nodeNetAddr, err
}

func (this *Transport) SetHostAddr(address common.Address, hostAddr string) {
	this.NodeAddressToIpPort.Store(address, hostAddr)
	this.NodeIpPortToAddress.Store(hostAddr, address)
}

func (this *Transport) StartHealthCheck(walletAddr common.Address) error {
	log.Infof("[StartHealthCheck] walletAddr: %s", common.ToBase58(walletAddr))

	var err error
	var nodeNetAddr string
	if v, ok := this.NodeAddressToIpPort.Load(walletAddr); ok {
		nodeNetAddr = v.(string)
	} else {
		if this.getHostAddrCallback == nil {
			err = fmt.Errorf("[StartHealthCheck] error: getHostAddrCallback is not set")
			log.Errorf("[StartHealthCheck] error:", err.Error())
			return err
		} else {
			nodeNetAddr, err = this.getHostAddrCallback(walletAddr)
			if err != nil {
				err = fmt.Errorf("[StartHealthCheck] getHostAddrCallback error: %s", err.Error())
				log.Errorf("[StartHealthCheck] error:", err.Error())
				return err
			}
		}
	}
	if nodeNetAddr == "" {
		err = fmt.Errorf("[StartHealthCheck] error: nodeNetAddr is nil string")
		log.Error("[StartHealthCheck] error:", err.Error())
		return err
	} else {
		client.P2pConnect(nodeNetAddr)
		return nil
	}
}

func (this *Transport) SetTargetQueueState(walletAddr common.Address, state int) {
	this.targetQueueState.Store(walletAddr, state)
}

func (this *Transport) GetTargetQueueState(walletAddr common.Address) int {
	queueState, exist := this.targetQueueState.Load(walletAddr)
	if exist {
		return queueState.(int)
	} else {
		return QueueNotExist
	}
}

func (this *Transport) GetNodeNetworkState(nodeAddr interface{}) string {
	var nodeNetAddress string
	switch nodeAddr.(type) {
	case string:
		nodeNetAddress = nodeAddr.(string)
	case common.Address:
		if v, ok := this.NodeAddressToIpPort.Load(nodeAddr); ok {
			nodeNetAddress = v.(string)
		} else {
			return transfer.NetworkUnknown
		}
	}

	state := client.GetNodeNetworkState(nodeNetAddress)
	nodeNetState := network.PeerState(state)
	switch nodeNetState {
	case network.PEER_UNKNOWN:
		return transfer.NetworkUnknown
	case network.PEER_UNREACHABLE:
		return transfer.NetworkUnreachable
	case network.PEER_REACHABLE:
		return transfer.NetworkReachable
	}
	return ""
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
