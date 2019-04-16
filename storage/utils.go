package storage

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/oniio/oniChain/common/log"
	"github.com/oniio/oniChannel/common"
	"github.com/oniio/oniChannel/common/constants"
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
source_statechange_id INTEGER NOT NULL, log_time TEXT, data JSON);`

const stateDbScriptCreateTables string = `PRAGMA foreign_keys=off;
BEGIN TRANSACTION;
%s%s%s
COMMIT;
PRAGMA foreign_keys=on;
`
const eventDbScriptCreateTables string = `PRAGMA foreign_keys=off;
BEGIN TRANSACTION;
%s%s
COMMIT;
PRAGMA foreign_keys=on;
`

func GetStateCreateTables() string {
	sqlStmt := fmt.Sprintf(stateDbScriptCreateTables, DbCreateSettings, DbCreateStateChanges, DbCreateSnapshot)

	return sqlStmt
}
func GetEventCreateTables() string {
	sqlStmt := fmt.Sprintf(eventDbScriptCreateTables, DbCreateSettings, DbCreateStateEvents)

	return sqlStmt
}

func formatByteSlicesForQuery(data []byte) string {
	value, _ := json.Marshal(data)
	valueStr := string(value)

	return valueStr[1 : len(valueStr)-1]
}
func formatAddressForQuery(address common.Address) string {
	str := address.String()
	return str[1 : len(str)-1]
}

func getValuesFromBalanceProof(data interface{}) (common.TokenAmount, common.TokenAmount, common.Locksroot) {
	var locksroot common.Locksroot

	event := reflect.ValueOf(data)
	balanceProofValue := reflect.Indirect(reflect.Indirect(event).FieldByName("BalanceProof"))

	transferredAmount := common.TokenAmount(balanceProofValue.FieldByName("TransferredAmount").Uint())
	lockedAmount := common.TokenAmount(balanceProofValue.FieldByName("LockedAmount").Uint())

	locksrootValue := balanceProofValue.FieldByName("LocksRoot")
	len := locksrootValue.Len()
	if len != constants.HASH_LEN {
		panic("locksroot length invalid")
	}

	for i := 0; i < len; i++ {
		locksroot[i] = byte(locksrootValue.Index(i).Uint())
	}

	return transferredAmount, lockedAmount, locksroot

}

func GetLatestKnownBalanceProofFromStateChanges(
	storage *SQLiteStorage, chainID common.ChainID, tokenNetworkID common.TokenNetworkID,
	channelIdentifier common.ChannelID, balanceHash common.BalanceHash, sender common.Address) *transfer.BalanceProofSignedState {

	//balanceHashStr := formatByteSlicesForQuery(balanceHash)
	//sendStr := formatAddressForQuery(sender)

	filters := map[string]interface{}{
		//"BalanceProof.ChainId":                 chainID,
		//"BalanceProof.TokenNetworkIdentifier ": tokenNetworkID,
		"BalanceProof.ChannelIdentifier": channelIdentifier,
		"BalanceProof.BalanceHash":       formatByteSlicesForQuery(balanceHash),
		"BalanceProof.Sender":            formatAddressForQuery(sender),
	}

	stateChangeRecord := storage.GetLatestStateChangeByDataField(filters)
	if stateChangeRecord.Data != nil {
		var balanceProof transfer.BalanceProofSignedState

		log.Debugf("[GetLatestKnownBalanceProofFromStateChanges] type: %s", reflect.ValueOf(stateChangeRecord.Data).String())
		balanceProof.TransferredAmount, balanceProof.LockedAmount, balanceProof.LocksRoot = getValuesFromBalanceProof(stateChangeRecord.Data)

		log.Debugf("[GetLatestKnownBalanceProofFromStateChanges] ta : %d, la : %d, lr: %v",
			balanceProof.TransferredAmount,
			balanceProof.LockedAmount,
			balanceProof.LocksRoot,
		)
		return &balanceProof
	}
	return nil
}

func GetLatestKnownBalanceProofFromEvents(
	storage *SQLiteStorage, chainID common.ChainID, tokenNetworkID common.TokenNetworkID,
	channelIdentifier common.ChannelID, balanceHash common.BalanceHash) *transfer.BalanceProofUnsignedState {

	//balanceHashStr := formatByteSlicesForQuery(balanceHash)
	filters := map[string]interface{}{
		//"BalanceProof.ChainId":                 chainID,
		//"BalanceProof.TokenNetworkIdentifier ": tokenNetworkID,
		"BalanceProof.ChannelIdentifier": channelIdentifier,
		"BalanceProof.BalanceHash":       formatByteSlicesForQuery(balanceHash),
	}

	eventRecord := storage.GetLatestEventByDataField(filters)
	if eventRecord.Data != nil {
		var balanceProof transfer.BalanceProofUnsignedState

		log.Debugf("[GetLatestKnownBalanceProofFromEvents] eventRecord.Data type: %s", reflect.ValueOf(eventRecord.Data).String())

		balanceProof.TransferredAmount, balanceProof.LockedAmount, balanceProof.LocksRoot = getValuesFromBalanceProof(eventRecord.Data)

		log.Debugf("[GetLatestKnownBalanceProofFromEvents] ta : %d, la : %d, lr: %v",
			balanceProof.TransferredAmount,
			balanceProof.LockedAmount,
			balanceProof.LocksRoot,
		)
		return &balanceProof
	}
	return nil
}

func GetStateChangeWithBalanceProofByLocksroot(
	storage *SQLiteStorage, chainID common.ChainID, tokenNetworkID common.TokenNetworkID,
	channelIdentifier common.ChannelID, locksroot common.Locksroot, sender common.Address) *StateChangeRecord {

	filters := map[string]interface{}{
		"BalanceProof.ChannelIdentifier": channelIdentifier,
		"BalanceProof.Locksroot":         locksroot,
	}

	return storage.GetLatestStateChangeByDataField(filters)
}

func GetEventWithBalanceProofByLocksroot(
	storage *SQLiteStorage, chainID common.ChainID, tokenNetworkID common.TokenNetworkID,
	channelIdentifier common.ChannelID, locksroot common.Locksroot) *EventRecord {

	filters := map[string]interface{}{
		"BalanceProof.ChannelIdentifier": channelIdentifier,
		"BalanceProof.Locksroot":         locksroot,
	}

	return storage.GetLatestEventByDataField(filters)
}
