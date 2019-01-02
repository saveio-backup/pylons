package utils

import (
	"container/list"
)

type NotifyingQueue struct {
	queue   list.List
	Maxsize int
}
