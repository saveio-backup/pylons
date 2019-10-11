package service

import (
	"encoding/hex"
	"fmt"

	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/transfer"
	"github.com/saveio/themis/common/log"
)

func GetBestRoutes(channelSrv *ChannelService, tokenNetworkId common.TokenNetworkID,
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

	tokenNetwork := transfer.GetTokenNetworkByIdentifier(channelSrv.StateFromChannel(), tokenNetworkId)
	if tokenNetwork == nil {
		log.Warnf("[GetBestRoutes] GetTokenNetworkByIdentifier error tokenNetwork is nil")
		return nil, fmt.Errorf("[GetBestRoutes] GetTokenNetworkByIdentifier error tokenNetwork is nil")
	}
	nodes := tokenNetwork.NetworkGraph.Nodes
	edges := tokenNetwork.NetworkGraph.Edges
	//log.Debug("[GetBestRoutes] edges", edges)

	top := transfer.NewTopology(nodes, edges)
	spt := top.GetShortPath(fromAddress)
	sptLen := len(spt)
	if len(spt) == 0 {
		log.Errorf("[GetBestRoutes] spt is nil")
		return nil, fmt.Errorf("[GetBestRoutes] spt is nil")
	}
	//log.Debugf("SPT:", spt)

	var partAddr common.Address
	var channelId common.ChannelID
	var i int
	for i = 0; i < sptLen; i++ {
		sp := spt[i]
		spLen := len(sp)
		if sp[spLen-1] == toAddress {
			partAddr = sp[0]
			networkState := channelSrv.GetNodeNetworkState(partAddr)
			if networkState == transfer.NetworkReachable {
				channelState := transfer.GetChannelStateByTokenNetworkAndPartner(channelSrv.StateFromChannel(),
					tokenNetworkId, partAddr)
				if channelState != nil {
					channelId = channelState.Identifier
					if valid, err := checkRouteAvailable(channelState, partAddr, fromAddress, amount); valid {
						log.Debugf("[GetBestRoutes]: %s", common.ToBase58(partAddr))
						break
					} else {
						log.Warnf("[GetBestRoutes] checkRouteAvailable %s error: ", common.ToBase58(partAddr), err.Error())
					}
				} else {
					log.Warnf("[GetBestRoutes] GetChannelStateByTokenNetworkAndPartner %s error", common.ToBase58(partAddr))
				}
			} else {
				log.Warnf("[GetBestRoutes] %s is NetworkUnReachable", common.ToBase58(partAddr))
			}
		}
	}
	if i == sptLen {
		log.Errorf("[GetBestRoutes] no route to target")
		return nil, fmt.Errorf("[GetBestRoutes] no route to target")
	}

	availableRoutes := []transfer.RouteState{{NodeAddress: partAddr, ChannelId: channelId}}
	return availableRoutes, nil
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
