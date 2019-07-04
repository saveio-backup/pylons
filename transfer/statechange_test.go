package transfer

import (
	"fmt"
	"github.com/saveio/pylons/common"
	"github.com/saveio/pylons/utils/jsonext"
	"testing"
)

func TestActionInitTargetMarshal(t *testing.T) {
	var initTarget1 ActionInitTarget
	initTarget1.Route = &RouteState{
		NodeAddress:       common.EmptyAddress,
		ChannelIdentifier: common.ChannelID(0),
	}

	balanceProof := &BalanceProofSignedState{
		Nonce:                  common.Nonce(0),
		TransferredAmount:      common.TokenAmount(0),
		LockedAmount:           common.TokenAmount(0),
		LocksRoot:              common.EmptySecretHash,
		TokenNetworkIdentifier: common.TokenNetworkID(common.EmptyTokenAddress),
		ChannelIdentifier:      common.ChannelID(0),
		MessageHash:            common.EmptySecretHash,
		Signature:              common.Signature{0x00},
		Sender:                 common.EmptyAddress,
		ChainId:                common.ChainID(0),
		PublicKey:              common.PubKey{0x00},
	}

	lock := &HashTimeLockState{
		Amount:     0,
		Expiration: 0,
		SecretHash: common.EmptySecretHash,
		Encoded:    []byte{0x01, 0x02, 0x03, 0x04},
		LockHash:   common.EmptySecretHash,
	}

	initTarget1.Transfer = &LockedTransferSignedState{
		MessageIdentifier: common.GetMsgID(),
		PaymentIdentifier: common.PaymentID(0),
		Token:             common.EmptyAddress,
		BalanceProof:      balanceProof,
		Lock:              lock,
		Initiator:         common.EmptyAddress,
		Target:            common.EmptyAddress,
	}

	data, err := jsonext.Marshal(initTarget1)
	if err != nil {
		t.Error(err.Error())
	}
	fmt.Println(data)

	//var initTarget2 *ActionInitTarget
	v, err := jsonext.UnmarshalExt(data, nil, CreateObjectByClassId)
	if err != nil {
		t.Error(err.Error())
	}
	if v == nil {
		t.Error("v is nil")
		return
	}
	//initTarget2 := v.(StateChange)
	initTarget2 := v.(*ActionInitTarget)
	fmt.Println(initTarget2)
}
