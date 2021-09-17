package service

import (
	"encoding/hex"
	"fmt"
	"github.com/saveio/pylons/route"

	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis/common/log"
)

func GetBestRoutesByDFS(channelSrv *ChannelService, tokenNetworkId common.TokenNetworkID, fromAddr common.Address,
	toAddr common.Address, amount common.TokenAmount, badAddrs []common.Address) ([]transfer.RouteState, error) {
	//""" Returns a list of channels that can be used to make a transfer.
	//
	//This will filter out channels that are not open and don't have enough capacity.
	//"""
	//# TODO: Route ranking.
	//# Rate each route to optimize the fee price/quality of each route and add a
	//# rate from in the range [0.0,1.0].

	tokenNetwork := transfer.GetTokenNetworkByIdentifier(channelSrv.StateFromChannel(), tokenNetworkId)
	if tokenNetwork == nil {
		log.Warnf("[GetBestRoutes] GetTokenNetworkByIdentifier error tokenNetwork is nil")
		return nil, fmt.Errorf("[GetBestRoutes] GetTokenNetworkByIdentifier error tokenNetwork is nil")
	}
	dnsAddrsMap := tokenNetwork.GetAllDns()
	if len(dnsAddrsMap) == 0 {
		dnsAddrsMap = tokenNetwork.GetAllDnsFromChain()
		if len(dnsAddrsMap) == 0 {
			return nil, fmt.Errorf("[GetBestRoutes] dnsAddrsMap is nil")
		}
	}
	nodes := tokenNetwork.NetworkGraph.Nodes
	edges := tokenNetwork.NetworkGraph.Edges
	routeDFS := &route.DFS{}
	routeDFS.NewTopology(nodes, edges, badAddrs)
	spt := routeDFS.GetShortPathTree(fromAddr, toAddr)
	sptLen := len(spt)
	if len(spt) == 0 {
		log.Errorf("[GetBestRoutes] spt is nil")
		return nil, fmt.Errorf("[GetBestRoutes] spt is nil")
	}
	log.Debugf("SPT:", sptLen)

	var nextHop common.Address
	var channelId common.ChannelID
	routeAvailable := false
	var isDnsNode bool
	for i := 0; i < sptLen; i++ {
		sp := spt[i]
		spLen := len(sp)
		if spLen < 2 {
			continue
		}
		// destination is not target
		if sp[0] != toAddr {
			continue
		}
		// [target ... media ... self]
		nextHop = sp[spLen-2]
		_, isDnsNode = dnsAddrsMap[nextHop]
		if nextHop != toAddr && !isDnsNode {
			log.Warnf("[GetBestRoutes] nextHop is not dns node: %s", common.ToBase58(nextHop))
			continue
		}
		networkState := channelSrv.GetNodeNetworkState(nextHop)
		if networkState != transfer.NetworkReachable {
			log.Warnf("[GetBestRoutes] %s is NetworkUnReachable", common.ToBase58(nextHop))
			continue
		}
		channelState := transfer.GetChannelStateByTokenNetworkAndPartner(channelSrv.StateFromChannel(),tokenNetworkId, nextHop)
		if channelState == nil {
			log.Warnf("[GetBestRoutes] GetChannelStateByTokenNetworkAndPartner %s error", common.ToBase58(nextHop))
			continue
		}
		channelId = channelState.Identifier
		if valid, err := checkRouteAvailable(channelState, nextHop, fromAddr, amount); valid {
			routeAvailable = true
			log.Debugf("[GetBestRoutes]: %s", common.ToBase58(nextHop))
			break
		} else {
			log.Warnf("[GetBestRoutes] checkRouteAvailable %s error: ", common.ToBase58(nextHop), err.Error())
		}
	}
	if !routeAvailable {
		log.Errorf("[GetBestRoutes] no route to target")
		return nil, fmt.Errorf("[GetBestRoutes] no route to target")
	}

	availableRoutes := []transfer.RouteState{{NodeAddress: nextHop, ChannelId: channelId}}
	return availableRoutes, nil
}

func GetBestRoutes(channelSrv *ChannelService, tokenNetworkId common.TokenNetworkID, fromAddr common.Address,
	toAddr common.Address, amount common.TokenAmount, badAddrs []common.Address) ([]transfer.RouteState, error) {
	//""" Returns a list of channels that can be used to make a transfer.
	//
	//This will filter out channels that are not open and don't have enough capacity.
	//"""
	//# Rate each route to optimize the fee price/quality of each route and add a
	//# rate from in the range [0.0,1.0].

	tokenNetwork := transfer.GetTokenNetworkByIdentifier(channelSrv.StateFromChannel(), tokenNetworkId)
	if tokenNetwork == nil {
		log.Warnf("[GetBestRoutes] GetTokenNetworkByIdentifier error tokenNetwork is nil")
		return nil, fmt.Errorf("[GetBestRoutes] GetTokenNetworkByIdentifier error tokenNetwork is nil")
	}
	dnsAddrsMap := tokenNetwork.GetAllDns()
	if len(dnsAddrsMap) == 0 {
		dnsAddrsMap = tokenNetwork.GetAllDnsFromChain()
		if len(dnsAddrsMap) == 0 {
			return nil, fmt.Errorf("[GetBestRoutes] dnsAddrsMap is nil")
		}
	}
	nodes := tokenNetwork.NetworkGraph.Nodes
	edges := tokenNetwork.NetworkGraph.Edges

	routeDijkstra := &route.Dijkstra{}
	for {
		routeDijkstra.NewTopology(nodes, edges, badAddrs)
		spt := routeDijkstra.GetShortPathTree(fromAddr, toAddr)
		sptLen := len(spt)
		if sptLen <= 0 {
			break
		}

		var nextHop common.Address
		var channelId common.ChannelID
		path := spt[0]
		// path may empty while not found route
		if len(path) < 2 {
			break
		}
		// [target ... media ... self]
		nextHop = path[len(path)-2]
		_, isDnsNode := dnsAddrsMap[nextHop]
		if nextHop != toAddr && !isDnsNode {
			badAddrs = append(badAddrs, nextHop)
			log.Warnf("[GetBestRoutes] nextHop is not dns node: %s", common.ToBase58(nextHop))
			continue
		}
		networkState := channelSrv.GetNodeNetworkState(nextHop)
		if networkState != transfer.NetworkReachable {
			badAddrs = append(badAddrs, nextHop)
			log.Warnf("[GetBestRoutes] %s is NetworkUnReachable", common.ToBase58(nextHop))
			continue
		}
		channelState := transfer.GetChannelStateByTokenNetworkAndPartner(channelSrv.StateFromChannel(),tokenNetworkId, nextHop)
		if channelState == nil {
			badAddrs = append(badAddrs, nextHop)
			log.Warnf("[GetBestRoutes] GetChannelStateByTokenNetworkAndPartner %s error", common.ToBase58(nextHop))
			continue
		}
		channelId = channelState.Identifier
		if valid, err := checkRouteAvailable(channelState, nextHop, fromAddr, amount); valid {
			availableRoutes := []transfer.RouteState{{NodeAddress: nextHop, ChannelId: channelId}}
			return availableRoutes, nil
		} else {
			badAddrs = append(badAddrs, nextHop)
			log.Warnf("[GetBestRoutes] checkRouteAvailable %s error %s: ", common.ToBase58(nextHop), err.Error())
		}
	}
	log.Errorf("[GetBestRoutes] no route to target")
	return nil, fmt.Errorf("[GetBestRoutes] no route to target")
}

func GetSpecifiedRoute(channelSrv *ChannelService, tokenNetworkId common.TokenNetworkID, media common.Address,
	fromAddress common.Address, amount common.TokenAmount) ([]transfer.RouteState, error) {
	var channelId common.ChannelID
	networkState := channelSrv.GetNodeNetworkState(media)
	if networkState == transfer.NetworkReachable {
		channelState := transfer.GetChannelStateByTokenNetworkAndPartner(channelSrv.StateFromChannel(),
			tokenNetworkId, media)
		if channelState != nil {
			channelId = channelState.Identifier
			if valid, err := checkRouteAvailable(channelState, media, fromAddress, amount); valid {
				log.Infof("[GetSpecifiedRoute] checkRouteAvailable %s valid", common.ToBase58(media))
			} else {
				log.Errorf("[GetBestRoutes] checkRouteAvailable %s error: %s", common.ToBase58(media), err.Error())
				return nil, fmt.Errorf("[GetBestRoutes] checkRouteAvailable %s error: %s", common.ToBase58(media), err.Error())
			}
		} else {
			log.Errorf("[GetBestRoutes] GetChannelStateByTokenNetworkAndPartner %s error", common.ToBase58(media))
			return nil, fmt.Errorf("[GetBestRoutes] GetChannelStateByTokenNetworkAndPartner %s error", common.ToBase58(media))
		}
	} else {
		log.Errorf("[GetBestRoutes] %s is NetworkUnReachable", common.ToBase58(media))
		return nil, fmt.Errorf("[GetBestRoutes] %s is NetworkUnReachable", common.ToBase58(media))
	}

	availableRoutes := []transfer.RouteState{{NodeAddress: media, ChannelId: channelId}}
	return availableRoutes, nil
}

func checkRouteAvailable(channelState *transfer.NettingChannelState, partAddr common.Address,
	fromAddress common.Address, amount common.TokenAmount) (bool, error) {
	if transfer.GetStatus(channelState) != transfer.ChannelStateOpened {
		return false, fmt.Errorf("channel is not opened, ignoring %s, %s ", hex.EncodeToString(fromAddress[:]),
			hex.EncodeToString(partAddr[:]))
	}

	distributable := transfer.GetDistributable(channelState.OurState, channelState.PartnerState)
	if amount > distributable {
		return false, fmt.Errorf("channel doesnt have enough funds, ignoring %s, %s, %d, %d ",
			hex.EncodeToString(fromAddress[:]), hex.EncodeToString(partAddr[:]), amount, distributable)
	}
	return true, nil
}
