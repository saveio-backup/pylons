package paymentproxy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"

	"github.com/oniio/oniChain/account"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChannel"
	"github.com/oniio/oniChannel/paymentproxy/messages"
	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
	"github.com/oniio/oniP2p/crypto/ed25519"
	"github.com/oniio/oniP2p/network"
	"github.com/oniio/oniP2p/network/addressmap"
	"github.com/oniio/oniP2p/types/opcode"
)

const (
	OPCODE_PAYMENT_PROXY opcode.Opcode = 2018 + iota
	OPCODE_BOOTSTRAP
	OPCODE_BOOTSTRAP_RESPONSE
)

const PROTOCOL = "tcp://"

var opcodes = map[opcode.Opcode]proto.Message{
	OPCODE_PAYMENT_PROXY:      &messages.PaymentProxy{},
	OPCODE_BOOTSTRAP:          &messages.BootStrap{},
	OPCODE_BOOTSTRAP_RESPONSE: &messages.BootStrapResponse{},
}

type addressItem struct {
	address  typing.Address
	isClient bool
}

type PaymentProxyService struct {
	net                             *network.Network
	channel                         *channel.Channel
	account                         *account.Account
	netAddress                      string
	mappingAddress                  string
	address                         typing.Address     // account address
	isClient                        bool               // if act as a router for payment
	bootstrap                       []string           //bootstap nodes ip addresses to connect to
	addressMap                      *sync.Map          // ip address -> account address
	initDeposit                     typing.TokenAmount // init Despoit for the channele with bootstrap
	notificationChannels            map[chan *transfer.EventPaymentReceivedSuccess]struct{}
	channelReceiveNotification      chan *transfer.EventPaymentReceivedSuccess
	nexthopToIndentifierToInitiator map[string]*sync.Map
}

func NewPaymentProxyService(channel *channel.Channel, netAddress string, mappingAddress string, isClient bool, bootstrap []string, initDeposit uint64) *PaymentProxyService {
	var address typing.Address

	if channel == nil {
		log.Error("invalid channel service")
		return nil
	}

	account := channel.Service.Account
	copy(address[:], account.Address[:20])

	for i, v := range bootstrap {
		bootstrap[i] = PROTOCOL + v
	}

	var mapAddress string

	if mappingAddress != "" {
		mapAddress = PROTOCOL + mappingAddress
	}

	paymentProxyService := &PaymentProxyService{
		channel:              channel,
		account:              account,
		address:              address,
		netAddress:           PROTOCOL + netAddress,
		mappingAddress:       mapAddress,
		isClient:             isClient,
		bootstrap:            bootstrap,
		addressMap:           new(sync.Map),
		initDeposit:          typing.TokenAmount(initDeposit),
		notificationChannels: make(map[chan *transfer.EventPaymentReceivedSuccess]struct{}),
	}

	// client proxy needs to register to channel for the payment notification
	if isClient {
		paymentProxyService.channelReceiveNotification = make(chan *transfer.EventPaymentReceivedSuccess)
		channel.RegisterReceiveNotification(paymentProxyService.channelReceiveNotification)

		paymentProxyService.nexthopToIndentifierToInitiator = make(map[string]*sync.Map)
	}

	return paymentProxyService
}

func (this *PaymentProxyService) Send(netAddress string, message proto.Message) error {
	signed, err := this.net.PrepareMessage(context.Background(), message)
	if err != nil {
		return fmt.Errorf("failed to sign message")
	}

	err = this.net.Write(netAddress, signed)
	if err != nil {
		return fmt.Errorf("failed to send message to %s", netAddress)
	}

	return nil
}

// from: the account from who we receie the message, it is not the initiator
func (this *PaymentProxyService) HandlePaymentProxyMessage(message *messages.PaymentProxy, from typing.Address) {
	initiator, _ := this.ConvertAddress(message.Initiator)
	target, _ := this.ConvertAddress(message.Target)

	// if client, it must be either the initiator or the target
	if this.isClient {
		log.Infof("Client HandlePaymentProxyMessage")
		if initiator == this.address {
			//pay the router
			this.DirectPay(message.Amount, from, message.PaymentID)
			return
		} else if target == this.address {
			// initiator want to pay us, but has no direct connection, so send a PaymentProxyMessage to connected router and pay it
			// find a route towards initiator and send the message
			nextHop := this.Route(initiator, from)
			this.Send(nextHop, message)

			// add to proxy cache so when a proxy payment is received from proxy, we know the initiator
			this.AddProxyMessageInitiatorCache(initiator, nextHop, typing.PaymentID(message.PaymentID))
			return
		} else {
			log.Error("receive a invalid PaymentProxy message")
			return
		}
	} else {
		log.Infof("Proxy HandlePaymentProxyMessage")
		var registryAddress typing.PaymentNetworkID
		var tokenAddress typing.TokenAddress

		// check if there is an channel with the message sender(not initiator)
		channelState := this.channel.Api.GetChannel(registryAddress, &tokenAddress, &from)
		if channelState == nil {
			log.Errorf("no payment channel can be found with partner : %v", from)
			return
		}

		contractBalance := channelState.OurState.ContractBalance

		// check if balance is enough
		balance := channelState.OurState.GetBalance()
		if (contractBalance - balance) < typing.TokenAmount(message.Amount) {
			log.Errorf("no enough balance to pay %s: amount = %d, contractBalance = %d, balance = %d", from, message.Amount, contractBalance, balance)
			return
		}

		nextHop := this.Route(initiator, from)

		// relay the message to next hop
		this.Send(nextHop, message)
		// pay for the sender
		this.DirectPay(message.Amount, from, message.PaymentID)
	}
}

func (this *PaymentProxyService) AddProxyMessageInitiatorCache(initiator typing.Address, nextHop string, identifier typing.PaymentID) {

	//paymentProxyService.nexthopToIndentifierToInitiator = make(map[string]*sync.Map)
	if !this.isClient {
		return
	}

	if paymentIdToInitiator, exist := this.nexthopToIndentifierToInitiator[nextHop]; exist {
		paymentIdToInitiator.Store(identifier, initiator)
	} else {
		paymentIdToInitiator := new(sync.Map)

		paymentIdToInitiator.Store(identifier, initiator)
		this.nexthopToIndentifierToInitiator[nextHop] = paymentIdToInitiator
	}
}

func (self *PaymentProxyService) GetInitiatorFromCache(nextHop string, identifier typing.PaymentID) (initiator typing.Address, ok bool) {
	var nilAddress typing.Address

	paymentIdToInitiator, exist := self.nexthopToIndentifierToInitiator[nextHop]
	if exist {
		initiator, ok := paymentIdToInitiator.Load(identifier)
		if ok {
			return initiator.(typing.Address), true
		}
	}

	return nilAddress, false
}

func (self *PaymentProxyService) RemoveInitiatorFromCache(nextHop string, identifier typing.PaymentID) (ok bool) {
	paymentIdToInitiator, exist := self.nexthopToIndentifierToInitiator[nextHop]
	if exist {
		_, ok := paymentIdToInitiator.Load(identifier)
		if ok {
			paymentIdToInitiator.Delete(identifier)
			return true
		}
	}

	return false
}

func (this *PaymentProxyService) DirectPay(amount uint64, target typing.Address, paymentID uint64) {
	log.Debugf("DirectPay to %s amount %d with paymentID %d", target, amount, paymentID)
	this.channel.Api.DirectTransferAsync(typing.TokenAmount(amount), target, typing.PaymentID(paymentID))
}

// check for possible route, return the netaddress of next hoop
func (this *PaymentProxyService) Route(target typing.Address, exclude typing.Address) (nextHop string) {
	// first round to check if there is a direct path to the target
	this.addressMap.Range(func(key, value interface{}) bool {
		netAddress := key.(string)
		item := value.(*addressItem)
		if item.isClient {
			// last hoop to the initiator
			if item.address == target && item.address != exclude {
				nextHop = netAddress
				return false
			}
		}
		return true
	})

	if nextHop != "" {
		return
	}

	// second round check the router for possible path
	this.addressMap.Range(func(key, value interface{}) bool {
		netAddress := key.(string)
		item := value.(*addressItem)
		//NOTE: with current requirement, there can only be one possilbe router connected
		// so retrun the address on the first router
		if !item.isClient && item.address != exclude {
			nextHop = netAddress
			return false
		}
		return true
	})

	return
}

func (this *PaymentProxyService) OnReceive(message proto.Message, from string) {

	switch message.(type) {
	case *messages.PaymentProxy:
		var isTarget bool
		var fromAddr typing.Address
		var err error

		msg := message.(*messages.PaymentProxy)

		// if client is the target, from address can not be obtained from Address map
		// but from the msg.Initiator
		if this.isClient {
			target, _ := this.ConvertAddress(msg.Target)
			if target == this.address {
				fromAddr, _ = this.ConvertAddress(msg.Initiator)
				isTarget = true
			}
		}
		// get sender account address
		// NOTE: msg.initiator is the origin of the payment
		if !isTarget {
			fromAddr, err = this.GetAddressFrmoAddressMap(from)
			if err != nil {
				log.Error(err)
				return
			}
		}

		this.HandlePaymentProxyMessage(msg, fromAddr)
		return
	case *messages.BootStrap:
		if this.isClient {
			log.Errorf("client recieves a bootstrap message from %s", from)
			return
		}

		log.Debugf("recieves a bootstrap message from %s", from)

		msg := message.(*messages.BootStrap)
		resp := &messages.BootStrapResponse{
			Sender:   this.address[:],
			IsClient: this.isClient,
			// dont fill the pubkey and signature now
		}

		err := this.Send(from, resp)
		if err != nil {
			log.Error(err)
		}

		this.UpdateAddressMap(msg.Sender, from, msg.IsClient)
		return
	case *messages.BootStrapResponse:
		log.Debugf("recieves a bootstrap response message from %s", from)

		msg := message.(*messages.BootStrapResponse)
		this.UpdateAddressMap(msg.Sender, from, msg.IsClient)

		// open channel between bootstrap nodes
		if !msg.IsClient && !this.isClient {
			this.InitPaymentChannelWithBootstrap(msg.Sender)
		}
		return
	}
}

func (this *PaymentProxyService) InitPaymentChannelWithBootstrap(partner []byte) {
	var registryAddress typing.PaymentNetworkID
	var tokenAddress typing.TokenAddress
	var settleTimeout typing.BlockTimeout
	var retryTimeout typing.NetworkTimeout

	settleTimeout = 5
	retryTimeout = 0.5

	partnerAddress, err := this.ConvertAddress(partner)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debugf("InitPaymentChannelWithBootstrap, try open channel with %s")
	this.channel.Api.ChannelOpen(registryAddress, tokenAddress, partnerAddress, settleTimeout, retryTimeout)
	this.channel.Api.SetTotalChannelDeposit(registryAddress, tokenAddress, partnerAddress, this.initDeposit, retryTimeout)
	log.Debugf("InitPaymentChannelWithBootstrap, set channel deposit %d success", this.initDeposit)
}

func (this *PaymentProxyService) GetNetAddressFromAddresMap(address typing.Address) (netAddress string, ok bool) {
	this.addressMap.Range(func(key, value interface{}) bool {
		item := value.(*addressItem)

		if item.address == address {
			netAddress = key.(string)
			return false
		}

		return true
	})

	if netAddress == "" {
		return "", false
	}

	return netAddress, true
}
func (this *PaymentProxyService) GetAddressFrmoAddressMap(netAddr string) (typing.Address, error) {
	var nilAddr typing.Address

	item, ok := this.addressMap.Load(netAddr)
	if ok {
		return (item.(*addressItem)).address, nil
	} else {
		return nilAddr, errors.New("No matching transport address found")
	}
}

func (this *PaymentProxyService) UpdateAddressMap(addr []byte, netAddr string, isClient bool) {
	address, err := this.ConvertAddress(addr)
	if err != nil {
		log.Error(err)
	}

	item := &addressItem{
		address:  address,
		isClient: isClient,
	}

	this.addressMap.LoadOrStore(netAddr, item)

	log.Debugf("update address mapping : %s : %v\n", netAddr, address)
	return
}

func (this *PaymentProxyService) ConvertAddress(addr []byte) (typing.Address, error) {
	var address typing.Address

	if len(addr) != typing.ADDR_LEN {
		return address, errors.New("invalid account address lenght")
	}

	copy(address[:], addr[:20])

	return address, nil
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
func (this *PaymentProxyService) Start() error {
	var err error

	builder := network.NewBuilderWithOptions(network.WriteFlushLatency(1 * time.Millisecond))

	builder.SetKeys(ed25519.RandomKeyPair())

	builder.SetAddress(this.netAddress)

	if this.mappingAddress != "" {
		builder.AddComponent(&addressmap.Component{MappingAddress: this.mappingAddress})
	}

	component := new(Component)
	component.Proxy = this
	builder.AddComponent(component)

	this.net, err = builder.Build()
	if err != nil {
		return err
	}

	err = registerMessages()
	if err != nil {
		return err
	}

	go this.net.Listen()

	this.net.BlockUntilListening()

	this.ConnectBootStrapNodes()

	if this.isClient {
		go this.ProcessNotificaion(this.channelReceiveNotification)
	}

	return nil
}

func (this *PaymentProxyService) ProcessNotificaion(receiveChan chan *transfer.EventPaymentReceivedSuccess) {
	var event *transfer.EventPaymentReceivedSuccess
	var networkIdentifier typing.PaymentNetworkID
	var tokenNetworkIdentifier typing.TokenNetworkID

	for {
		select {
		case event = <-receiveChan:
			var nextHop string

			// get nextHop by searching with event.Initiator in the addressMap
			nextHop, ok := this.GetNetAddressFromAddresMap(typing.Address(event.Initiator))
			if !ok {
				continue
			}

			initiator, ok := this.GetInitiatorFromCache(nextHop, event.Identifier)
			if !ok {
				continue
			}

			newEvent := &transfer.EventPaymentReceivedSuccess{
				PaymentNetworkIdentifier: networkIdentifier,
				TokenNetworkIdentifier:   tokenNetworkIdentifier,
				Identifier:               event.Identifier,
				Amount:                   event.Amount,
				Initiator:                typing.InitiatorAddress(initiator),
			}

			this.RemoveInitiatorFromCache(nextHop, event.Identifier)

			// notify the client with new event
			for channel := range this.notificationChannels {
				channel <- newEvent
			}
		}
	}
}

// connect to bootstrap node
func (this *PaymentProxyService) ConnectBootStrapNodes() {
	message := &messages.BootStrap{
		Sender:   this.account.Address[:],
		IsClient: this.isClient,
		// dont fill the publickey and signature now
	}

	// connect to bootstrap nodes
	this.net.Bootstrap(this.bootstrap...)

	for _, netAddress := range this.bootstrap {
		// send bootstrap messsage to bootstrap nodes
		this.Send(netAddress, message)
	}
}

// called by upper layer to start a proxied payment
func (this *PaymentProxyService) InitSendPaymentProxy(targetNetAddress string, initiator typing.Address, target typing.Address, amount typing.TokenAmount, paymentID typing.PaymentID) error {
	if !this.isClient {
		return errors.New(" InitSendPaymentProxy must be called by a client")
	}

	time.Sleep(5 * time.Millisecond)

	log.Debugf("InitSendPaymentProxy")
	message := &messages.PaymentProxy{
		Initiator: initiator[:],
		Target:    target[:],
		Amount:    uint64(amount),
		PaymentID: uint64(paymentID),
	}

	fullAddress := PROTOCOL + targetNetAddress

	this.net.Bootstrap(fullAddress)

	err := this.Send(fullAddress, message)
	return err
}

func (this *PaymentProxyService) RegisterProxyReceiveNotification(notificaitonChannel chan *transfer.EventPaymentReceivedSuccess) {
	this.notificationChannels[notificaitonChannel] = struct{}{}
}
