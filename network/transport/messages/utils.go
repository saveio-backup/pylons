package messages

import (
	"github.com/oniio/oniChannel/common"
)

func ConvertAddress(addr *Address) common.Address {
	var address common.Address

	copy(address[:], addr.Address[:20])
	return address
}

func ConvertLocksroot(lock *Locksroot) common.Locksroot {
	var locksroot common.Locksroot

	copy(locksroot[:], lock.Locksroot[:32])
	return locksroot
}
