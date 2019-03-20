package channelservice

import (
	"encoding/hex"
	"fmt"

	chainComm "github.com/oniio/oniChain/common"
	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/transfer"
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

	toAddr := chainComm.Address(toAddress)
	toNode := toAddr.ToBase58()
	fromAddr := chainComm.Address(fromAddress)
	frBase58Addr := fromAddr.ToBase58()

	tokenNetwork := transfer.GetTokenNetworkByIdentifier(chainState, tokenNetworkId)
	networkStatuses := transfer.GetNetworkStatuses(chainState)

	nodes := tokenNetwork.NetworkGraph.Nodes
	edges := tokenNetwork.NetworkGraph.Edges
	log.Debug("[GetBestRoutes] edges", edges)


	top := transfer.NewTopology(nodes, edges)
	spt := top.GetShortPath(frBase58Addr)
	sptLen := len(spt)
	if len(spt) == 0 {
		log.Errorf("[GetBestRoutes] spt is nil")
		return nil, fmt.Errorf("[GetBestRoutes] spt is nil")
	}
	log.Debug("SPT:", spt)

/*

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
*/
	var i int
	for i = 0; i < sptLen; i++ {
		sp := spt[i]
		spLen := len(sp)
		if sp[spLen - 1] == toNode {
			break
		}
	}
	base58PartAddr := spt[i][0]
	partnerAddress, err := chainComm.AddressFromBase58(base58PartAddr)
	if err != nil {
		log.Errorf("[GetBestRoutes] error: %v", err.Error())
		return nil, fmt.Errorf("[GetBestRoutes] error: %v", err.Error())
	}

	partAddr := common.Address(partnerAddress)
	channelState := transfer.GetChannelStateByTokenNetworkAndPartner(chainState, tokenNetworkId, partAddr)
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
	networkState := (*networkStatuses)[partAddr] //, NODE_NETWORK_UNKNOWN)
	log.Debug("networkState:   ", networkState)
	//if networkState != common.NodeNetworkReachable {
	//	return nil, fmt.Errorf("partner for channel state isn't reachable, ignoring  %s, %s, %s ",
	//		hex.EncodeToString(fromAddress[:]), hex.EncodeToString(partnerAddress[:]), networkState)
	//}
	routeState := transfer.RouteState{
		NodeAddress:       partAddr,
		ChannelIdentifier: channelState.Identifier,
	}

	var availableRoutes []transfer.RouteState
	availableRoutes = append(availableRoutes, routeState)
	return availableRoutes, nil
}
