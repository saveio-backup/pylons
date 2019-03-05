package storage

import (
	"container/list"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/utils/jsonext"
)

const ChannelDbVersion int = 6

type EventRecord struct {
	EventRecordIdentifier int
	StateChangeIdentifier int
	Data                  transfer.Event
}

type StateChangeRecord struct {
	stateChnangeIdentifier int
	Data                   transfer.StateChange
}

func assertSqliteVersion() bool {
	return true
}

type SQLiteStorage struct {
	connState      *sql.DB
	connEvent      *sql.DB
	writeStateLock sync.Mutex
	writeEventLock sync.Mutex
	StateSync      sync.WaitGroup
	EventSync      sync.WaitGroup
}

func NewSQLiteStorage(databasePath string) (*SQLiteStorage, error) {
	self := new(SQLiteStorage)
	i := strings.LastIndex(databasePath, ".")
	if i < 0 {
		return nil, errors.New(`new sqlite database error: can't find "."`)
	}
	statePath := string(databasePath[0:i]) + "-state.db"
	//state
	connState, err := sql.Open("sqlite3", statePath)
	if err != nil {
		return nil, err
	}

	_, err = connState.Exec("PRAGMA synchronous = NORMAL")
	if err != nil {
		return nil, err
	}

	_, err = connState.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	self.connState = connState

	createStateScript := GetStateCreateTables()
	_, err = self.connState.Exec(createStateScript)
	if err != nil {
		return nil, err
	}

	//events
	evnetPath := string(databasePath[0:i]) + "-event.db"
	connEvent, err := sql.Open("sqlite3", evnetPath)
	if err != nil {
		return nil, err
	}

	_, err = connEvent.Exec("PRAGMA synchronous = NORMAL")
	if err != nil {
		return nil, err
	}

	_, err = connEvent.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	self.connEvent = connEvent

	createEventScript := GetEventCreateTables()
	_, err = self.connEvent.Exec(createEventScript)
	if err != nil {
		return nil, err
	}

	self.runUpdates()

	return self, nil
}

func (self *SQLiteStorage) runUpdates() (bool, error) {

	stmtState, err := self.connState.Prepare("INSERT OR REPLACE INTO settings(name, value) VALUES(?, ?)")
	if err != nil {
		return false, err
	}

	_, err = stmtState.Exec("version", ChannelDbVersion)
	if err != nil {
		return false, err
	}
	stmtState.Close()
	stmtEvent, err := self.connEvent.Prepare("INSERT OR REPLACE INTO settings(name, value) VALUES(?, ?)")
	if err != nil {
		return false, err
	}

	_, err = stmtEvent.Exec("version", ChannelDbVersion)
	if err != nil {
		return false, err
	}
	stmtEvent.Close()
	return true, nil
}

func (self *SQLiteStorage) getVersion() int {
	var version int

	version = ChannelDbVersion

	stmt, err := self.connState.Prepare("SELECT value FROM settings WHERE name=?")
	if err != nil {
		return ChannelDbVersion
	}
	defer stmt.Close()

	rows, err := stmt.Query("version")
	if err != nil {
		return ChannelDbVersion
	}
	defer rows.Close()

	var versionStr string
	for rows.Next() {
		err = rows.Scan(&versionStr)
		if err != nil {
			return ChannelDbVersion
		}
		version, err = strconv.Atoi(versionStr)
		break
	}

	return version
}

func (self *SQLiteStorage) writeStateChange(stateChange transfer.StateChange, stateChangeId *int) {
	serializedData, _ := jsonext.Marshal(stateChange)
	self.StateSync.Add(1)
	self.writeStateLock.Lock()
	defer self.writeStateLock.Unlock()

	stmt, err := self.connState.Prepare("INSERT INTO state_changes(data) VALUES(?)")
	if err != nil {
		log.Fatalln(err)
	}
	defer stmt.Close()

	sqlRes, _ := stmt.Exec(serializedData)

	lastRowId, _ := sqlRes.LastInsertId()

	*stateChangeId = int(lastRowId)

}

func (self *SQLiteStorage) writeStateSnapshot(stateChangeId int, snapshot transfer.State) int {
	serializedData, _ := jsonext.Marshal(snapshot)

	self.writeStateLock.Lock()
	defer self.writeStateLock.Unlock()

	stmt, _ := self.connState.Prepare("INSERT INTO state_snapshot(statechange_id, data) VALUES(?, ?)")
	defer stmt.Close()
	sqlRes, _ := stmt.Exec(stateChangeId, serializedData)

	lastRowId, _ := sqlRes.LastInsertId()
	return int(lastRowId)
}

func (self *SQLiteStorage) writeEvents(stateChangeId int, events []transfer.Event, logTime string) {
	self.EventSync.Add(1)
	self.writeEventLock.Lock()
	defer self.writeEventLock.Unlock()

	stmt, err := self.connEvent.Prepare("INSERT INTO state_events(source_statechange_id, log_time, data) VALUES(?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for _, e := range events {
		serializedData, _ := jsonext.Marshal(e)
		stmt.Exec(stateChangeId, logTime, serializedData)
	}
	self.EventSync.Done()
	return
}

func (self *SQLiteStorage) getLatestStateSnapshot() (int, transfer.State) {
	self.writeStateLock.Lock()
	defer self.writeStateLock.Unlock()
	rows, _ := self.connState.Query("SELECT statechange_id, data from state_snapshot ORDER BY identifier DESC LIMIT 1")
	defer rows.Close()

	var lastAppliedStateChangeId int
	var snapshotData []byte

	for rows.Next() {
		err := rows.Scan(&lastAppliedStateChangeId, &snapshotData)
		if err == nil {
			v, _ := jsonext.UnmarshalExt(snapshotData, nil, transfer.CreateObjectByClassId)

			if snapshotState, ok := v.(transfer.State); ok {
				return lastAppliedStateChangeId, snapshotState
			} else {
				return 0, nil
			}
		}
	}

	return 0, nil
}

func (self *SQLiteStorage) getSnapshotClosestToStateChange(stateChangeId interface{}) (int, transfer.State) {
	var rows *sql.Rows
	var realStateChangeId int
	self.writeStateLock.Lock()
	defer self.writeStateLock.Unlock()
	latest := false
	switch stateChangeId.(type) {
	case string:
		latest = true
	case int:
		realStateChangeId = stateChangeId.(int)
	}

	if latest == true {
		rows, _ = self.connState.Query("SELECT identifier FROM state_changes ORDER BY identifier DESC LIMIT 1")

		for rows.Next() {
			rows.Scan(&realStateChangeId)
			break
		}
		rows.Close()
	}

	rows, _ = self.connState.Query("SELECT statechange_id, data FROM state_snapshot WHERE statechange_id <= ? ORDER BY identifier DESC LIMIT 1", realStateChangeId)
	defer rows.Close()

	var lastAppliedStateChangeId int
	var snapshotData []byte

	for rows.Next() {
		err := rows.Scan(&lastAppliedStateChangeId, &snapshotData)
		if err == nil {
			v, _ := jsonext.UnmarshalExt(snapshotData, nil, transfer.CreateObjectByClassId)
			if snapshotState, ok := v.(transfer.State); ok {
				return lastAppliedStateChangeId, snapshotState
			} else {
				return 0, nil
			}
		}
	}

	return lastAppliedStateChangeId, nil
}

func (self *SQLiteStorage) GetLatestEventByDataField(filters map[string]interface{}) *EventRecord {

	var finalWhereClause string
	var rows *sql.Rows
	self.writeEventLock.Lock()
	defer self.writeEventLock.Unlock()
	first := true

	len := len(filters)
	fields := []string{}
	args := []interface{}{}

	for k, v := range filters {
		if first == false {
			finalWhereClause = fmt.Sprintf("%v AND json_extract(data, ?)=? ", finalWhereClause)
		} else {
			first = false
			finalWhereClause = fmt.Sprintf("json_extract(data, ?)=? ")
		}

		realFiled := fmt.Sprintf("$.%v", k)
		fields = append(fields, realFiled)
		args = append(args, v)
	}

	finalQuerySql := fmt.Sprintf("%v%v%v", "SELECT identifier, source_statechange_id, data FROM state_events WHERE ",
		finalWhereClause, "ORDER BY identifier DESC LIMIT 1")

	stmt, _ := self.connEvent.Prepare(finalQuerySql)
	defer stmt.Close()

	switch len {
	case 1:
		rows, _ = stmt.Query(fields[0], args[0])
	case 2:
		rows, _ = stmt.Query(fields[0], args[0], fields[1], args[1])
	case 3:
		rows, _ = stmt.Query(fields[0], args[0], fields[1], args[1], fields[2], args[2])
	}
	defer rows.Close()

	var eventId, stateChangeId int
	var snapshotData []byte

	for rows.Next() {
		err := rows.Scan(&eventId, &stateChangeId, &snapshotData)
		if err == nil {
			v, err := jsonext.UnmarshalExt(snapshotData, nil, transfer.CreateObjectByClassId)
			if err != nil {
				return &EventRecord{}
			}
			if latestEvent, ok := v.(transfer.Event); ok {
				return &EventRecord{eventId, stateChangeId, latestEvent}
			} else {
				return &EventRecord{}
			}
		}
	}

	return &EventRecord{}
}

func (self *SQLiteStorage) GetLatestEventsByDataField(filters map[string]interface{}) *list.List {

	var finalWhereClause string
	var rows *sql.Rows

	first := true

	len := len(filters)
	fields := []string{}
	args := []interface{}{}

	for k, v := range filters {
		if first == false {
			finalWhereClause = fmt.Sprintf("%v AND json_extract(data, ?)=? ", finalWhereClause)
		} else {
			first = false
			finalWhereClause = fmt.Sprintf("json_extract(data, ?)=? ")
		}

		realFiled := fmt.Sprintf("$.%v", k)
		fields = append(fields, realFiled)
		args = append(args, v)
	}

	finalQuerySql := fmt.Sprintf("%v%v%v", "SELECT identifier, source_statechange_id, data FROM state_events WHERE ",
		finalWhereClause, "ORDER BY identifier DESC")
	self.writeEventLock.Lock()
	defer self.writeEventLock.Unlock()
	stmt, _ := self.connEvent.Prepare(finalQuerySql)
	defer stmt.Close()

	switch len {
	case 1:
		rows, _ = stmt.Query(fields[0], args[0])
	case 2:
		rows, _ = stmt.Query(fields[0], args[0], fields[1], args[1])
	case 3:
		rows, _ = stmt.Query(fields[0], args[0], fields[1], args[1], fields[2], args[2])
	}
	defer rows.Close()

	var eventId, stateChangeId int
	var snapshotData []byte
	result := list.New()

	for rows.Next() {
		err := rows.Scan(&eventId, &stateChangeId, &snapshotData)
		if err == nil {
			v, err := jsonext.UnmarshalExt(snapshotData, nil, transfer.CreateObjectByClassId)
			if err != nil {
				return result
			}
			if latestEvent, ok := v.(transfer.Event); ok {
				event := &EventRecord{eventId, stateChangeId, latestEvent}
				result.PushBack(event)
			} else {
				return result
			}
		}
	}

	return result
}

func (self *SQLiteStorage) GetLatestStateChangeByDataField(filters map[string]interface{}) *StateChangeRecord {
	var rows *sql.Rows
	var finalWhereClause string

	first := true

	len := len(filters)
	fields := []string{}
	args := []interface{}{}

	for k, v := range filters {
		if first == false {
			finalWhereClause = fmt.Sprintf("%v AND json_extract(data, ?)=? ", finalWhereClause)
		} else {
			first = false
			finalWhereClause = fmt.Sprintf("json_extract(data, ?)=? ")
		}

		realFiled := fmt.Sprintf("$.%v", k)
		fields = append(fields, realFiled)
		args = append(args, v)
	}

	finalQuerySql := fmt.Sprintf("%v%v%v", "SELECT identifier, data FROM state_changes WHERE ",
		finalWhereClause, "ORDER BY identifier DESC LIMIT 1")
	self.writeStateLock.Lock()
	defer self.writeStateLock.Unlock()
	stmt, _ := self.connState.Prepare(finalQuerySql)
	defer stmt.Close()

	switch len {
	case 1:
		rows, _ = stmt.Query(fields[0], args[0])
	case 2:
		rows, _ = stmt.Query(fields[0], args[0], fields[1], args[1])
	case 3:
		rows, _ = stmt.Query(fields[0], args[0], fields[1], args[1], fields[2], args[2])
	}
	defer rows.Close()

	var stateChangeId int
	var stateChangetData []byte

	for rows.Next() {
		err := rows.Scan(&stateChangeId, &stateChangetData)
		if err == nil {
			v, err := jsonext.UnmarshalExt(stateChangetData, nil, transfer.CreateObjectByClassId)
			if err != nil {
				return &StateChangeRecord{}
			}
			if latestStateChange, ok := v.(transfer.StateChange); ok {
				return &StateChangeRecord{stateChangeId, latestStateChange}
			} else {
				return &StateChangeRecord{}
			}
		}
	}
	return &StateChangeRecord{}
}

func (self *SQLiteStorage) GetLatestStateChangesByDataField(filters map[string]interface{}) *list.List {
	var rows *sql.Rows
	var finalWhereClause string

	first := true

	len := len(filters)
	fields := []string{}
	args := []interface{}{}

	for k, v := range filters {
		if first == false {
			finalWhereClause = fmt.Sprintf("%v AND json_extract(data, ?)=? ", finalWhereClause)
		} else {
			first = false
			finalWhereClause = fmt.Sprintf("json_extract(data, ?)=? ")
		}

		realFiled := fmt.Sprintf("$.%v", k)
		fields = append(fields, realFiled)
		args = append(args, v)
	}

	finalQuerySql := fmt.Sprintf("%v%v%v", "SELECT identifier, data FROM state_changes WHERE ",
		finalWhereClause, "ORDER BY identifier DESC")
	self.writeStateLock.Lock()
	defer self.writeStateLock.Unlock()
	stmt, _ := self.connState.Prepare(finalQuerySql)
	defer stmt.Close()

	switch len {
	case 1:
		rows, _ = stmt.Query(fields[0], args[0])
	case 2:
		rows, _ = stmt.Query(fields[0], args[0], fields[1], args[1])
	case 3:
		rows, _ = stmt.Query(fields[0], args[0], fields[1], args[1], fields[2], args[2])
	}
	defer rows.Close()

	var stateChangeId int
	var stateChangetData []byte
	result := list.New()

	for rows.Next() {
		err := rows.Scan(&stateChangeId, &stateChangetData)
		if err == nil {
			v, err := jsonext.UnmarshalExt(stateChangetData, nil, transfer.CreateObjectByClassId)
			if err != nil {
				return result
			}
			if latestStateChange, ok := v.(transfer.StateChange); ok {
				stateChange := &StateChangeRecord{stateChangeId, latestStateChange}
				result.PushBack(stateChange)
			} else {
				return result
			}
		}
	}
	return result
}

func (self *SQLiteStorage) getStateChangesByIdentifier(fromIdentifier interface{},
	toIdentifier interface{}) *list.List {

	var rows *sql.Rows
	var realFromIdentifier, realToIdentifier int

	fromLatest := false
	switch fromIdentifier.(type) {
	case string:
		fromLatest = true
	case int:
		realFromIdentifier = fromIdentifier.(int)
	}
	self.writeStateLock.Lock()
	defer self.writeStateLock.Unlock()
	if fromLatest == true {
		rows, _ = self.connState.Query("SELECT identifier FROM state_changes ORDER BY identifier DESC LIMIT 1")

		for rows.Next() {
			rows.Scan(&realFromIdentifier)
			break
		}
		rows.Close()
	}

	toLatest := false
	switch toIdentifier.(type) {
	case string:
		toLatest = true
	case int:
		realToIdentifier = toIdentifier.(int)
	}

	if toLatest == true {
		rows, _ = self.connState.Query("SELECT data FROM state_changes WHERE identifier >= ?", realFromIdentifier)
	} else {
		rows, _ = self.connState.Query("SELECT data FROM state_changes WHERE identifier BETWEEN ? AND ?", realFromIdentifier, realToIdentifier)
	}
	defer rows.Close()

	var stateChangeData []byte

	result := list.New()
	for rows.Next() {
		err := rows.Scan(&stateChangeData)
		if err == nil {
			v, err := jsonext.UnmarshalExt(stateChangeData, nil, transfer.CreateObjectByClassId)
			if err != nil {
				continue
			}
			if stateChange, ok := v.(transfer.StateChange); ok {
				result.PushBack(stateChange)
			}
		}
	}

	return result
}

func (self *SQLiteStorage) queryEvents(limit int, offset int) *sql.Rows {
	rows, _ := self.connEvent.Query("SELECT data, log_time FROM state_events ORDER BY identifier ASC LIMIT ? OFFSET ?", limit, offset)
	return rows
}

func (self *SQLiteStorage) GetEventsWithTimestamps(limit int, offset int) *list.List {
	result := list.New()
	self.writeEventLock.Lock()
	defer self.writeEventLock.Unlock()
	rows := self.queryEvents(limit, offset)
	if rows == nil {
		return result
	} else {
		var eventData []byte
		var timestamp string

		for rows.Next() {
			err := rows.Scan(&eventData, &timestamp)
			if err == nil {
				v, err := jsonext.UnmarshalExt(eventData, nil, transfer.CreateObjectByClassId)
				if err != nil {
					continue
				}
				if event, ok := v.(transfer.Event); ok {
					timestampEvent := &TimestampedEvent{event, timestamp}
					result.PushBack(timestampEvent)
				}
			}
		}
		rows.Close()
	}

	return result
}

func (self *SQLiteStorage) GetEvents(limit int, offset int) *list.List {
	result := list.New()
	self.writeEventLock.Lock()
	defer self.writeEventLock.Unlock()
	rows := self.queryEvents(limit, offset)
	if rows == nil {
		return result
	} else {
		var eventData []byte

		for rows.Next() {
			err := rows.Scan(&eventData)
			if err == nil {
				v, err := jsonext.UnmarshalExt(eventData, nil, transfer.CreateObjectByClassId)
				if err != nil {
					continue
				}
				if event, ok := v.(transfer.Event); ok {
					result.PushBack(event)
				}
			}
		}
		rows.Close()
	}

	return result
}

func (self *SQLiteStorage) Close() {
	self.connState.Close()
	self.connEvent.Close()
}
