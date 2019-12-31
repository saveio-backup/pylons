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
	"github.com/saveio/pylons/network/transport/messages"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis/common/log"
)

const MAX_RETRY = 5

type ChannelServiceInterface interface {
	OnMessage(proto.Message, string)
	Sign(message interface{}) error
	HandleStateChange(stateChange transfer.StateChange) []transfer.Event
	StateFromChannel() *transfer.ChainState
	GetAllMessageQueues() transfer.QueueIdsToQueuesType
}

type Transport struct {
	queueLock           *sync.Mutex
	messageQueues       map[transfer.QueueId]*Queue
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
		queueLock:       new(sync.Mutex),
		kill:            make(chan struct{}),
		messageQueues:   make(map[transfer.QueueId]*Queue),
		addressQueueMap: new(sync.Map),
		ChannelService:  channelService,
	}
}

// messages first be queued, only can be send when Delivered for previous msssage is received
func (this *Transport) SendAsync(queueId *transfer.QueueId, msg proto.Message) error {
	var err error
	var msgID *messages.MessageID

	switch msg.(type) {
	case *messages.DirectTransfer:
		msgID = (msg.(*messages.DirectTransfer)).MessageId
	case *messages.Processed:
		msgID = (msg.(*messages.Processed)).MessageId
	case *messages.LockedTransfer:
		msgID = (msg.(*messages.LockedTransfer)).BaseMessage.MessageId
	case *messages.SecretRequest:
		msgID = (msg.(*messages.SecretRequest)).MessageId
	case *messages.RevealSecret:
		msgID = (msg.(*messages.RevealSecret)).MessageId
	case *messages.RefundTransfer:
		msgID = (msg.(*messages.RefundTransfer)).Refund.BaseMessage.MessageId
	case *messages.BalanceProof:
		msgID = (msg.(*messages.BalanceProof)).MessageId
	case *messages.LockExpired:
		msgID = (msg.(*messages.LockExpired)).MessageId
	case *messages.WithdrawRequest:
		msgID = (msg.(*messages.WithdrawRequest)).MessageId
	case *messages.Withdraw:
		msgID = (msg.(*messages.Withdraw)).MessageId
	case *messages.CooperativeSettleRequest:
		msgID = (msg.(*messages.CooperativeSettleRequest)).MessageId
	case *messages.CooperativeSettle:
		msgID = (msg.(*messages.CooperativeSettle)).MessageId
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

func (this *Transport) GetQueue(queueId *transfer.QueueId) *Queue {
	this.queueLock.Lock()
	defer this.queueLock.Unlock()
	q, ok := this.messageQueues[*queueId]
	if !ok {
		q = this.initQueue(queueId)
	}
	return q
}

func (this *Transport) QueueSend(queue *Queue, queueId transfer.QueueId) {
	var interval time.Duration = 10
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
				}
				retryTimes = 0
			} else {
				log.Debug("[DeliverChan] msgId.MessageId != item.messageId.MessageId queue.Len: ", queue.Len())
				log.Warnf("[DeliverChan] MessageId not match (%d  %d)", msgId.MessageId, item.messageId.MessageId)
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

func (this *Transport) CheckIfNeedRemove(queueId *transfer.QueueId, item *QueueItem) bool {
	messageId := common.MessageID(item.messageId.MessageId)
	queues := this.ChannelService.GetAllMessageQueues()

	// check if queue exist
	if events, exist := queues[*queueId]; exist {
		// check if messageid in queue
		for _, event := range events {
			message := transfer.GetSenderMessageEvent(event)
			if message.MessageId == messageId {
				return false
			}
		}
	}
	return true
}

func (this *Transport) PeekAndSend(queue *Queue, queueId *transfer.QueueId) error {
	item, ok := queue.Peek()
	if !ok {
		return fmt.Errorf("Error peeking from queue. ")
	}

	msg := item.(*QueueItem).message
	msgId := common.MessageID(item.(*QueueItem).messageId.MessageId)

	if this.CheckIfNeedRemove(queueId, item.(*QueueItem)) {
		log.Debugf("remove msg msg = %+v\n", msg)
		queue.DeliverChan <- &messages.MessageID{MessageId: uint64(msgId)}
		return nil
	}

	log.Debugf("send msg msg = %+v\n", msg)
	address, err := this.GetHostAddr(queueId.Recipient)
	if err != nil {
		log.Errorf("[PeekAndSend] GetHostAddr %s, error: %s", common.ToBase58(queueId.Recipient), err.Error())
		return errors.New("no valid address to send message")
	}

	log.Debugf("[PeekAndSend] address: %s msgId: %v, queue: %p, len: %d\n", address, msgId, queue, queue.Len())

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

func (this *Transport) GetNodeNetworkState(nodeAddr common.Address) string {
	var err error
	var nodeNetAddress string

	if this.getHostAddrCallback == nil {
		log.Errorf("[GetNodeNetworkState] getHostAddrCallback is not set!!!")
		panic("[GetNodeNetworkState] getHostAddrCallback is not set!!!")
	}
	if nodeNetAddress, err = this.getHostAddrCallback(nodeAddr); err != nil {
		log.Errorf("[GetNodeNetworkState] getHostAddrCallback err: %s", err.Error())
		return transfer.NetworkUnknown
	}

	state := client.GetNodeNetworkState(nodeNetAddress)
	nodeNetState := network.PeerState(state)
	switch nodeNetState {
	case network.PEER_UNKNOWN:
		log.Errorf("[GetNodeNetworkState] nodeNetAddress: %s is unknown", nodeNetAddress)
		return transfer.NetworkUnknown
	case network.PEER_UNREACHABLE:
		log.Errorf("[GetNodeNetworkState] nodeNetAddress: %s is unreachable", nodeNetAddress)
		return transfer.NetworkUnreachable
	case network.PEER_REACHABLE:
		return transfer.NetworkReachable
	}
	return ""
}

func (this *Transport) SetGetHostAddrCallback(getHostAddrCallback func(address common.Address) (string, error)) {
	this.getHostAddrCallback = getHostAddrCallback
}

func (this *Transport) GetHostAddrByCallBack(walletAddr common.Address) (string, error) {
	if this.getHostAddrCallback == nil {
		log.Errorf("[GetHostAddrByCallBack] getHostAddrCallback is not set!!!")
		panic("[GetHostAddrByCallBack] getHostAddrCallback is not set!!!")
	}

	nodeNetAddr, err := this.getHostAddrCallback(walletAddr)
	if err == nil {
		if client.GetNodeNetworkState(nodeNetAddr) == int(network.PEER_REACHABLE) {
			return nodeNetAddr, nil
		} else {
			return "", fmt.Errorf("[GetHostAddrByCallBack] %s is unreachable", common.ToBase58(walletAddr))
		}
	} else {
		return "", fmt.Errorf("[GetHostAddrByCallBack] %s error: %s", common.ToBase58(walletAddr), err.Error())
	}
}

func (this *Transport) GetHostAddr(walletAddr common.Address) (string, error) {
	nodeNetAddr, err := this.GetHostAddrByCallBack(walletAddr)
	if err != nil {
		log.Errorf("[GetHostAddr] GetHostAddrByCallBack: %s error: %s", common.ToBase58(walletAddr), err.Error())
	}
	return nodeNetAddr, err
}

func (this *Transport) StartHealthCheck(walletAddr common.Address) error {
	log.Debugf("[StartHealthCheck] walletAddr: %s", common.ToBase58(walletAddr))

	var err error
	var nodeNetAddr string

	if this.getHostAddrCallback == nil {
		log.Errorf("[StartHealthCheck] getHostAddrCallback is not set!!!")
		panic("[StartHealthCheck] getHostAddrCallback is not set!!!")
	}

	nodeNetAddr, err = this.getHostAddrCallback(walletAddr)
	if err != nil {
		err = fmt.Errorf("[StartHealthCheck] getHostAddrCallback error: %s", err.Error())
		log.Errorf("[StartHealthCheck] error:", err.Error())
		return err
	}
	if nodeNetAddr == "" {
		err = fmt.Errorf("[StartHealthCheck] error: nodeNetAddr is nil string")
		log.Error("[StartHealthCheck] error:", err.Error())
		return err
	} else {
		return client.P2pConnect(nodeNetAddr)
	}
}

func (this *Transport) Receive(message proto.Message, from string) {
	//log.Info("[NetComponent] Receive: ", reflect.TypeOf(message).String(), " From: ", from)

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
		msgID = msg.MessageId
	case *messages.Processed:
		msg := message.(*messages.Processed)
		senderWallerAddr = messages.ConvertAddress(msg.Signature.Sender)
		msgID = msg.MessageId
	case *messages.LockedTransfer:
		msg := message.(*messages.LockedTransfer)
		senderWallerAddr = messages.ConvertAddress(msg.BaseMessage.EnvelopeMessage.Signature.Sender)
		msgID = msg.BaseMessage.MessageId
	case *messages.LockExpired:
		msg := message.(*messages.LockExpired)
		senderWallerAddr = messages.ConvertAddress(msg.EnvelopeMessage.Signature.Sender)
		msgID = msg.MessageId
	case *messages.RefundTransfer:
		msg := message.(*messages.RefundTransfer)
		senderWallerAddr = messages.ConvertAddress(msg.Refund.BaseMessage.EnvelopeMessage.Signature.Sender)
		msgID = msg.Refund.BaseMessage.MessageId
	case *messages.SecretRequest:
		msg := message.(*messages.SecretRequest)
		senderWallerAddr = messages.ConvertAddress(msg.Signature.Sender)
		msgID = msg.MessageId
	case *messages.RevealSecret:
		msg := message.(*messages.RevealSecret)
		senderWallerAddr = messages.ConvertAddress(msg.Signature.Sender)
		msgID = msg.MessageId
	case *messages.BalanceProof:
		msg := message.(*messages.BalanceProof)
		senderWallerAddr = messages.ConvertAddress(msg.EnvelopeMessage.Signature.Sender)
		msgID = msg.MessageId
	case *messages.WithdrawRequest:
		msg := message.(*messages.WithdrawRequest)
		senderWallerAddr = messages.ConvertAddress(msg.Participant)
		msgID = msg.MessageId
	case *messages.Withdraw:
		msg := message.(*messages.Withdraw)
		senderWallerAddr = messages.ConvertAddress(msg.PartnerSignature.Sender)
		msgID = msg.MessageId
	case *messages.CooperativeSettleRequest:
		msg := message.(*messages.CooperativeSettleRequest)
		senderWallerAddr = messages.ConvertAddress(msg.Participant1Signature.Sender)
		msgID = msg.MessageId
	case *messages.CooperativeSettle:
		msg := message.(*messages.CooperativeSettle)
		senderWallerAddr = messages.ConvertAddress(msg.Participant2Signature.Sender)
		msgID = msg.MessageId
	default:
		log.Warn("[ReceiveMessage] unknown Msg type: ", reflect.TypeOf(message).String())
		return
	}

	log.Debugf("[ReceiveMessage] %s msgId: %d fromNetAddr: %s fromWalletAddr: %s", reflect.TypeOf(message).String(),
		msgID.MessageId, fromNetAddr, common.ToBase58(senderWallerAddr))
	deliveredMessage := &messages.Delivered{
		DeliveredMessageId: msgID,
	}

	//var nodeNetAddress string
	err := this.ChannelService.Sign(deliveredMessage)
	if err == nil {
		log.Debugf("[SendDeliver] (%v) MessageId: %d to:  %s", reflect.TypeOf(message).String(),
			deliveredMessage.DeliveredMessageId.MessageId, fromNetAddr)

		senderNetAddr, err := this.GetHostAddr(senderWallerAddr)
		if err != nil {
			log.Errorf("[PeekAndSend] state != NetworkReachable %s", senderNetAddr)
		}

		if err = client.P2pSend(senderNetAddr, deliveredMessage); err != nil {
			log.Errorf("[SendDeliver] (%v) MessageId: %d to: %s error: %s",
				reflect.TypeOf(message).String(), deliveredMessage.DeliveredMessageId.MessageId,
				senderNetAddr, err.Error())
		}
	} else {
		log.Errorf("[SendDeliver] (%v) deliveredMessage Sign error: ", err.Error(),
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
	msgId := common.MessageID(msg.DeliveredMessageId.MessageId)
	queue, ok := this.addressQueueMap.Load(msgId)
	if !ok {
		log.Debugf("[ReceiveDelivered] from: %s Time: %s msgId: %v\n", from, time.Now().String(), msgId)
		log.Error("[ReceiveDelivered] msg.DeliveredMessageId is not in addressQueueMap")
		return
	}
	log.Debugf("[ReceiveDelivered] from: %s Time: %s msgId: %v, queue: %p\n", from, time.Now().String(), msgId, queue)
	queue.(*Queue).DeliverChan <- msg.DeliveredMessageId
}

func (this *Transport) initQueue(queueId *transfer.QueueId) *Queue {
	q := NewQueue(uint32(common.Config.MaxMsgQueue))
	this.messageQueues[*queueId] = q

	// queueId cannot be pointer type otherwise it might be updated outside QueueSend
	go this.QueueSend(q, *queueId)
	return q
}
