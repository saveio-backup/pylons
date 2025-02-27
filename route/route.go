package route

import (
	"github.com/saveio/pylons/common"
	"sync"
)

type ShortPathTree [][]common.Address

func (s ShortPathTree) Len() int {
	return len(s)
}
func (s ShortPathTree) Less(i, j int) bool {
	return len(s[i]) < len(s[j])
}

func (s ShortPathTree) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type route interface {
	NewTopology(nodes, edges *sync.Map, blacklist []common.Address)
	// GetShortPathTree return spt should contains path with format: [target ... media ... self]
	GetShortPathTree(from, to common.Address) ShortPathTree
}

