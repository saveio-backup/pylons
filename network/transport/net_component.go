package transport

import (
	"github.com/saveio/pylons/network/transport/messages"
	"github.com/saveio/carrier/network"
)

type NetComponent struct {
	*network.Component
	Net *P2pNetwork
}

func (this *NetComponent) Startup(net *network.Network) {
}

func (this *NetComponent) Receive(ctx *network.ComponentContext) error {
	msg := ctx.Message()
	addr := ctx.Client().Address

	switch msg.(type) {
	case *messages.Processed:
		this.Net.Receive(msg, addr)
	case *messages.Delivered:
		this.Net.Receive(msg, addr)
	case *messages.SecretRequest:
		this.Net.Receive(msg, addr)
	case *messages.Secret:
		this.Net.Receive(msg, addr)
	case *messages.RevealSecret:
		this.Net.Receive(msg, addr)
	case *messages.DirectTransfer:
		this.Net.Receive(msg, addr)
	case *messages.LockedTransfer:
		this.Net.Receive(msg, addr)
	case *messages.RefundTransfer:
		this.Net.Receive(msg, addr)
	case *messages.LockExpired:
		this.Net.Receive(msg, addr)
	case *messages.WithdrawRequest:
		this.Net.Receive(msg, addr)
	case *messages.Withdraw:
		this.Net.Receive(msg, addr)
	case *messages.CooperativeSettleRequest:
		this.Net.Receive(msg, addr)
	case *messages.CooperativeSettle:
		this.Net.Receive(msg, addr)
	}
	return nil
}
