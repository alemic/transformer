package store

import (
	"expvar"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/jmhodges/levigo"
)

type levelDbWriteMode bool

const (
	LevelDbReadOnly  levelDbWriteMode = true
	LevelDbReadWrite levelDbWriteMode = false
)

var recordsRead, bytesRead, recordsWritten, bytesWritten, seeks *expvar.Int

func init() {
	recordsRead = expvar.NewInt("RecordsRead")
	recordsWritten = expvar.NewInt("RecordsWritten")
	bytesRead = expvar.NewInt("BytesRead")
	bytesWritten = expvar.NewInt("BytesWritten")
	seeks = expvar.NewInt("Seeks")
}

type LevelDbStore struct {
	dbPath       string
	writeMode    bool
	dbOpenLock   sync.Mutex
	readIterator *levigo.Iterator
	readOptions  *levigo.ReadOptions
	writeOptions *levigo.WriteOptions
	db           *levigo.DB
	dbOpts       *levigo.Options
}

// Create a DatastoreFull that can read and write to a LevelDB database.
// Connections to this database are on-demand, so the database isn't locked
// until you BeginReading or BeginWriting.
func NewLevelDbStore(dbPath string, writeMode levelDbWriteMode) *LevelDbStore {
	return &LevelDbStore{
		dbPath:    dbPath,
		writeMode: bool(writeMode),
	}
}

func (store *LevelDbStore) openDatabase() error {
	if store.readOptions != nil || store.writeOptions != nil {
		return nil
	}
	dbOpts := levigo.NewOptions()
	dbOpts.SetMaxOpenFiles(128)
	dbOpts.SetCreateIfMissing(!store.writeMode)
	dbOpts.SetBlockSize(1 << 22) // 4 MB
	db, err := levigo.Open(store.dbPath, dbOpts)
	if err != nil {
		dbOpts.Close()
		return err
	}
	store.db = db
	store.dbOpts = dbOpts
	return nil
}

func (store *LevelDbStore) closeDatabase() {
	if store.readOptions != nil && store.writeOptions != nil {
		return
	}
	store.db.Close()
	store.dbOpts.Close()
}

func (store *LevelDbStore) BeginReading() error {
	store.dbOpenLock.Lock()
	defer store.dbOpenLock.Unlock()
	if store.readOptions != nil {
		panic("Only one routine may read from a LevelDB at a time.")
	}
	if err := store.openDatabase(); err != nil {
		return err
	}
	store.readOptions = levigo.NewReadOptions()
	store.readIterator = store.db.NewIterator(store.readOptions)
	store.readIterator.SeekToFirst()
	return nil
}

func (store *LevelDbStore) ReadRecord() (*Record, error) {
	if !store.readIterator.Valid() {
		return nil, store.readIterator.GetError()
	}

	record := &Record{
		Key:   store.readIterator.Key(),
		Value: store.readIterator.Value(),
	}
	recordsRead.Add(1)
	bytesRead.Add(int64(len(record.Key) + len(record.Value)))
	store.readIterator.Next()
	return record, nil
}

func (store *LevelDbStore) EndReading() error {
	store.dbOpenLock.Lock()
	defer store.dbOpenLock.Unlock()
	store.readOptions.Close()
	store.readIterator.Close()
	store.closeDatabase()
	store.readOptions = nil
	return nil
}

func (store *LevelDbStore) BeginWriting() error {
	store.dbOpenLock.Lock()
	defer store.dbOpenLock.Unlock()
	if store.writeOptions != nil {
		panic("Only one routine may write to a LevelDB at a time.")
	}
	if err := store.openDatabase(); err != nil {
		return err
	}
	store.writeOptions = levigo.NewWriteOptions()
	return nil
}

func (store *LevelDbStore) WriteRecord(record *Record) error {
	if err := store.db.Put(store.writeOptions, record.Key, record.Value); err != nil {
		return fmt.Errorf("Error writing to database: %v", err)
	}
	recordsWritten.Add(1)
	bytesWritten.Add(int64(len(record.Key) + len(record.Value)))
	return nil
}

func (store *LevelDbStore) EndWriting() error {
	store.dbOpenLock.Lock()
	defer store.dbOpenLock.Unlock()
	store.writeOptions.Close()
	store.closeDatabase()
	store.writeOptions = nil
	return nil
}

func (store *LevelDbStore) Seek(key []byte) error {
	if store.readOptions == nil {
		panic("You may only seek while reading")
	}
	store.readIterator.Seek(key)
	return nil
}

func (store *LevelDbStore) DeleteAllRecords() error {
	store.dbOpenLock.Lock()
	defer store.dbOpenLock.Unlock()

	if store.readOptions == nil && store.writeOptions == nil {
		panic("You may only call DeleteAllRecords after starting reading or writing")
	}

	writeOptions := store.writeOptions
	if writeOptions == nil {
		writeOptions = levigo.NewWriteOptions()
		defer writeOptions.Close()
	}
	readOptions := store.readOptions
	if readOptions == nil {
		readOptions = levigo.NewReadOptions()
		defer readOptions.Close()
	}
	it := store.db.NewIterator(readOptions)
	defer it.Close()
	it.SeekToFirst()
	for ; it.Valid(); it.Next() {
		if err := store.db.Delete(writeOptions, it.Key()); err != nil {
			return fmt.Errorf("Error clearing keys from database: %v", err)
		}
	}
	if err := it.GetError(); err != nil {
		return fmt.Errorf("Error iterating through database: %v", err)
	}
	return nil
}

type levelDbManager string

// Manage a set of LevelDB databases in the provided directory.
//
// The LevelDB constructor methods for the returned Manager take a single
// parameter, the name of a LevelDB inside the dbRoot.
func NewLevelDbManager(dbRoot string) Manager {
	return levelDbManager(dbRoot)
}

func (dirname levelDbManager) open(writeMode levelDbWriteMode, params ...interface{}) *LevelDbStore {
	if len(params) != 1 {
		panic(fmt.Errorf("NewLevelDbStore accepts a single argument. the path of the store"))
	}
	basename := params[0].(string)
	filename := filepath.Join(string(dirname), basename)
	return NewLevelDbStore(filename, writeMode)
}

func (m levelDbManager) Reader(params ...interface{}) Reader {
	return m.open(LevelDbReadOnly, params...)
}
func (m levelDbManager) Writer(params ...interface{}) Writer {
	return m.open(LevelDbReadWrite, params...)
}
func (m levelDbManager) Seeker(params ...interface{}) Seeker {
	return m.open(LevelDbReadOnly, params...)
}
func (m levelDbManager) Deleter(params ...interface{}) Deleter {
	return m.open(LevelDbReadWrite, params...)
}
func (m levelDbManager) ReadingWriter(params ...interface{}) ReadingWriter {
	return m.open(LevelDbReadWrite, params...)
}
func (m levelDbManager) SeekingWriter(params ...interface{}) SeekingWriter {
	return m.open(LevelDbReadWrite, params...)
}
func (m levelDbManager) ReadingDeleter(params ...interface{}) ReadingDeleter {
	return m.open(LevelDbReadWrite, params...)
}
func (m levelDbManager) SeekingDeleter(params ...interface{}) SeekingDeleter {
	return m.open(LevelDbReadWrite, params...)
}
