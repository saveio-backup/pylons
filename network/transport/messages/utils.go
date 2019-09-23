package messages

import (
	"github.com/saveio/pylons/common"
)

func ConvertAddress(addr *Address) common.Address {
	var address common.Address

	copy(address[:], addr.Address[:20])
	return address
}

func ConvertLocksRoot(lock *LocksRoot) common.LocksRoot {
	var locksRoot common.LocksRoot

	copy(locksRoot[:], lock.LocksRoot[:32])
	return locksRoot
}
