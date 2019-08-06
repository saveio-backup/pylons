package network

import (
	"github.com/saveio/carrier/network"
)

type NetComponent struct {
	*network.Component
	Net *Network
}

func (this *NetComponent) Startup(net *network.Network) {
}

func (this *NetComponent) Receive(ctx *network.ComponentContext) error {
	msg := ctx.Message()
	addr := ctx.Client().Address
	return this.Net.Receive(ctx, msg, addr)
}
