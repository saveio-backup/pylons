package channelservice

import (
	"encoding/hex"
	"fmt"

	chainComm "github.com/oniio/oniChain/common"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/transfer"
	"github.com/daseinio/x-dsp/log"
)

func GetBestRoutes(chainState *transfer.ChainState, tokenNetworkId common.TokenNetworkID,
	fromAddress common.Address, toAddress common.Address, amount common.TokenAmount,
	previousAddress common.Address) ([]transfer.RouteState, error) {

	//""" Returns a list of channels that can be used to make a transfer.
	//
	//This will filter out channels that are not open and don't have enough
	//capacity.
	//"""
	//# TODO: Route ranking.
	//# Rate each route to optimize the fee price/quality of each route and add a
	//# rate from in the range [0.0,1.0].

	var availableRoutes []transfer.RouteState

	tokenNetwork := transfer.GetTokenNetworkByIdentifier(chainState, tokenNetworkId)
	networkStatuses := transfer.GetNetworkStatuses(chainState)

	nodes := tokenNetwork.NetworkGraph.Nodes
	edges := tokenNetwork.NetworkGraph.Edges
	log.Debug("[GetBestRoutes] edges", edges)

	top := transfer.NewTopology(nodes, edges)
	toAddr := chainComm.Address(toAddress)
	toNode := transfer.Node{Name: toAddr.ToBase58()}
	spt := top.SPT(toNode)

	fromAddr := chainComm.Address(fromAddress)
	frBase58Addr := fromAddr.ToBase58()
	//fromNode := dijkstra.Node{Name: frBase58Addr}

	var shortPath []transfer.Edge
	var partnerAddress common.Address
	for fr, path := range spt {
		if frBase58Addr == fr.Name {
			shortPath = path.Edges
		}
	}

	for _, edge := range shortPath {
		node1, node2, _ := edge.NodeA, edge.NodeB, edge.Distance
		if node1.Name == frBase58Addr {
			partAddress, err := chainComm.AddressFromBase58(node2.Name)
			if err != nil {
				continue
			}
			partnerAddress = common.Address(partAddress)
			break
		}
		if node2.Name == frBase58Addr {
			partAddress, err := chainComm.AddressFromBase58(node1.Name)
			if err != nil {
				continue
			}
			partnerAddress = common.Address(partAddress)
			break
		}
	}

	partAddr := chainComm.Address(partnerAddress)
	partBase58Addr := partAddr.ToBase58()
	log.Debug("PartBase58Addr: ", partBase58Addr)

	channelState := transfer.GetChannelStateByTokenNetworkAndPartner(chainState, tokenNetworkId, partnerAddress)
	if channelState == nil {
		return nil, fmt.Errorf("GetChannelStateByTokenNetworkAndPartner error")
	}

	if transfer.GetStatus(channelState) != transfer.ChannelStateOpened {
		return nil, fmt.Errorf("channel is not opened, ignoring %s, %s ", hex.EncodeToString(fromAddress[:]),
			hex.EncodeToString(partnerAddress[:]))
	}
	distributable := transfer.GetDistributable(channelState.OurState, channelState.PartnerState)
	if amount > distributable {
		return nil, fmt.Errorf("channel doesnt have enough funds, ignoring %s, %s, %d, %d ", hex.EncodeToString(fromAddress[:]),
			hex.EncodeToString(partnerAddress[:]), amount, distributable)
	}
	networkState := (*networkStatuses)[partnerAddress] //, NODE_NETWORK_UNKNOWN)
	log.Debug("networkState:   ", networkState)
	//if networkState != common.NodeNetworkReachable {
	//	return nil, fmt.Errorf("partner for channel state isn't reachable, ignoring  %s, %s, %s ",
	//		hex.EncodeToString(fromAddress[:]), hex.EncodeToString(partnerAddress[:]), networkState)
	//}
	routeState := transfer.RouteState{
		NodeAddress:       partnerAddress,
		ChannelIdentifier: channelState.Identifier,
	}

	availableRoutes = append(availableRoutes, routeState)
	return availableRoutes, nil
}
