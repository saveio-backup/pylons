package storage

import (
	"fmt"

	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/transfer"
)

type TimestampedEvent struct {
	WrappedEvent transfer.Event
	logTime      string
}

const databasePath string = "./channel.db"

const DbCreateSettings string = "CREATE TABLE IF NOT EXISTS settings (name VARCHAR[24] NOT NULL PRIMARY KEY, value TEXT);"
const DbCreateStateChanges string = "CREATE TABLE IF NOT EXISTS state_changes (identifier INTEGER PRIMARY KEY AUTOINCREMENT, data JSON);"
const DbCreateSnapshot string = `CREATE TABLE IF NOT EXISTS state_snapshot (identifier INTEGER PRIMARY KEY, statechange_id INTEGER,
data JSON, FOREIGN KEY(statechange_id) REFERENCES state_changes(identifier));`

const DbCreateStateEvents string = `CREATE TABLE IF NOT EXISTS state_events (identifier INTEGER PRIMARY KEY,
source_statechange_id INTEGER NOT NULL, log_time TEXT, data JSON, FOREIGN KEY(source_statechange_id) REFERENCES state_changes(identifier));`

const DbScriptCreateTables string = `PRAGMA foreign_keys=off;
BEGIN TRANSACTION;
%s%s%s%s
COMMIT;
PRAGMA foreign_keys=on;
`

func GetDbScriptCreateTables() string {
	sqlStmt := fmt.Sprintf(DbScriptCreateTables, DbCreateSettings, DbCreateStateChanges, DbCreateSnapshot, DbCreateStateEvents)

	return sqlStmt
}

func GetLatestKnownBalanceProofFromStateChanges(
	storage *SQLiteStorage, chainID common.ChainID, tokenNetworkID common.TokenNetworkID,
	channelIdentifier common.ChannelID, balanceHash common.BalanceHash, sender common.Address) *transfer.BalanceProofSignedState {

	type BalanceProofSignedState struct {
		Nonce                  common.Nonce
		TransferredAmount      common.TokenAmount
		LockedAmount           common.TokenAmount
		LocksRoot              common.Locksroot
		TokenNetworkIdentifier common.TokenNetworkID
		ChannelIdentifier      common.ChannelID
		MessageHash            common.Keccak256
		Signature              common.Signature
		Sender                 common.Address
		ChainId                common.ChainID
		PublicKey              common.PubKey
	}

	filters := map[string]interface{}{
		//"BalanceProof.ChainId":                 chainID,
		//"BalanceProof.TokenNetworkIdentifier ": tokenNetworkID,
		"BalanceProof.ChannelIdentifier": channelIdentifier,
		//"BalanceProof.Sender":            sender,
	}

	stateChangeRecord := storage.GetLatestStateChangeByDataField(filters)

	if stateChangeRecord.Data != nil {

		switch stateChangeRecord.Data.(type) {
		case *transfer.ReceiveTransferDirect:
			stateChange := stateChangeRecord.Data.(*transfer.ReceiveTransferDirect)

			balanceProof := stateChange.BalanceProof
			if balanceProof != nil && common.AddressEqual(balanceProof.Sender, sender) {
				hash := transfer.HashBalanceData(
					balanceProof.TransferredAmount,
					balanceProof.LockedAmount,
					balanceProof.LocksRoot,
				)

				if common.SliceEqual(hash, balanceHash) {
					return balanceProof
				}
			}
			return nil
		}
	}

	return nil
}

func GetLatestKnownBalanceProofFromEvents(
	storage *SQLiteStorage, chainID common.ChainID, tokenNetworkID common.TokenNetworkID,
	channelIdentifier common.ChannelID, balanceHash common.BalanceHash) *transfer.BalanceProofUnsignedState {

	filters := map[string]interface{}{
		//"BalanceProof.ChainId":                 chainID,
		//"BalanceProof.TokenNetworkIdentifier ": tokenNetworkID,
		"BalanceProof.ChannelIdentifier": channelIdentifier,
	}

	eventRecord := storage.GetLatestEventByDataField(filters)

	if eventRecord.Data != nil {
		switch eventRecord.Data.(type) {
		case *transfer.SendDirectTransfer:
			event := eventRecord.Data.(*transfer.SendDirectTransfer)

			balanceProof := event.BalanceProof
			if balanceProof != nil {
				hash := transfer.HashBalanceData(
					balanceProof.TransferredAmount,
					balanceProof.LockedAmount,
					balanceProof.LocksRoot,
				)

				if common.SliceEqual(hash, balanceHash) {
					return balanceProof
				}
			}
			return nil
		}
	}

	return nil
}
