package transfer

import (
	"bytes"

	"errors"
	"fmt"
	"strconv"

	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/common/constants"
)

type QueueId struct {
	Recipient common.Address
	ChannelId common.ChannelID
}

func (self QueueId) String() string {
	return fmt.Sprintf("%v-%v", self.Recipient, self.ChannelId)
}

func (self QueueId) MarshalText() (text []byte, err error) {
	var scratch [64]byte
	var e bytes.Buffer

	e.WriteByte('[')
	for i := 0; i < constants.AddrLen; i++ {
		b := strconv.AppendUint(scratch[:0], uint64(self.Recipient[i]), 10)
		e.Write(b)
		if i < constants.AddrLen-1 {
			e.WriteByte(' ')
		}

	}
	e.WriteByte(']')

	b := strconv.AppendInt(scratch[:0], int64(self.ChannelId), 10)
	e.Write(b)

	return e.Bytes(), nil
}

func (self *QueueId) UnmarshalText(text []byte) error {
	newText := text[1:]

	startIdx := 0
	for i := 0; i < constants.AddrLen; i++ {
		for newText[startIdx] == ' ' || newText[startIdx] == '[' {
			startIdx++
		}

		toIdx := startIdx
		for newText[toIdx] >= '0' && newText[toIdx] <= '9' {
			toIdx++
		}

		res, err := strconv.ParseUint(string(newText[startIdx:toIdx]), 10, 8)

		if err != nil {
			return errors.New("QueueId TextUnmarshaler error!")
		} else {
			self.Recipient[i] = byte(res)
		}

		startIdx = toIdx
	}

	for newText[startIdx] == ' ' || newText[startIdx] == ']' {
		startIdx++
	}

	res, err := strconv.ParseInt(string(newText[startIdx:]), 10, 32)

	if err != nil {
		return errors.New("QueueId TextUnmarshaler error!")
	} else {
		self.ChannelId = common.ChannelID(res)
	}

	return nil
}
