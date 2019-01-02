package paymentproxy

import (
	"github.com/oniio/oniChannel/paymentproxy/messages"
	"github.com/oniio/oniP2p/network"
)

type Component struct {
	*network.Component
	Proxy *PaymentProxyService
}

func (this *Component) Startup(net *network.Network) {
}

func (this *Component) Receive(ctx *network.ComponentContext) error {
	msg := ctx.Message()
	addr := ctx.Client().Address

	switch msg.(type) {
	case *messages.PaymentProxy:
		this.Proxy.OnReceive(msg, addr)
	case *messages.BootStrap:
		this.Proxy.OnReceive(msg, addr)
	case *messages.BootStrapResponse:
		this.Proxy.OnReceive(msg, addr)
	}
	return nil
}
