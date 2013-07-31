package store

import (
	"bytes"
	"log"
	"sort"
)

// A SliceStore is a simple DatastoreFull that keeps its records in memory. It
// is suitable for testing and small data sets, but should not be used for
// larger data. Use LevelDbStore for larger data sets.
type SliceStore struct {
	records []*Record
	cursor  int
}

type recordSlice []*Record

func (p recordSlice) Len() int           { return len(p) }
func (p recordSlice) Less(i, j int) bool { return bytes.Compare(p[i].Key, p[j].Key) < 0 }
func (p recordSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (store *SliceStore) BeginReading() error {
	sort.Sort(recordSlice(store.records))
	store.cursor = -1
	return nil
}

func (store *SliceStore) ReadRecord() (*Record, error) {
	store.cursor++
	if store.cursor >= len(store.records) {
		return nil, nil
	}
	return store.records[store.cursor].Copy(), nil
}

func (store *SliceStore) EndReading() error {
	return nil
}

func (store *SliceStore) BeginWriting() error {
	return nil
}

func (store *SliceStore) WriteRecord(record *Record) error {
	for idx, existingRecord := range store.records {
		if bytes.Equal(record.Key, existingRecord.Key) {
			store.records[idx] = record.Copy()
			return nil
		}
	}
	store.records = append(store.records, record.Copy())
	return nil
}

func (store *SliceStore) EndWriting() error {
	return nil
}

func (store *SliceStore) DeleteAllRecords() error {
	store.records = nil
	return nil
}

func (store *SliceStore) Seek(key []byte) error {
	store.cursor = -1
	for store.cursor < len(store.records) {
		if store.cursor+1 >= len(store.records) || bytes.Compare(store.records[store.cursor+1].Key, key) >= 0 {
			break
		}
		store.cursor++
	}
	return nil
}

func (store *SliceStore) Print() {
	store.BeginReading()
	for {
		record, err := store.ReadRecord()
		if err != nil {
			panic(err)
		}
		if record == nil {
			break
		}
		log.Printf("%s: %s (%v: %v)", record.Key, record.Value, record.Key, record.Value)
	}
}
