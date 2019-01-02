package messages

import (
	"github.com/oniio/oniChannel/typing"
)

func ConvertAddress(addr *Address) typing.Address {
	var address typing.Address

	copy(address[:], addr.Address[:20])
	return address
}

func ConvertLocksroot(lock *Locksroot) typing.Locksroot {
	var locksroot typing.Locksroot

	copy(locksroot[:], lock.Locksroot[:32])
	return locksroot
}
