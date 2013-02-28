package transformer

import (
	"fmt"
)

func makeLevelDbRecord(key string, value string, databaseIndex uint8) *LevelDbRecord {
	return &LevelDbRecord{
		Key:           []byte(key),
		Value:         []byte(value),
		DatabaseIndex: databaseIndex,
	}
}

func ExampleDemuxInputsSorted() {
	firstStore := SliceStore{}
	firstStore.BeginWriting()
	firstStore.WriteRecord(makeLevelDbRecord("d", "foo0", 0))
	firstStore.WriteRecord(makeLevelDbRecord("f", "bar0", 0))
	firstStore.WriteRecord(makeLevelDbRecord("h", "baz0", 0))
	firstStore.EndWriting()

	secondStore := SliceStore{}
	secondStore.BeginWriting()
	secondStore.WriteRecord(makeLevelDbRecord("e", "foo1", 1))
	secondStore.WriteRecord(makeLevelDbRecord("g", "bar1", 1))
	secondStore.WriteRecord(makeLevelDbRecord("i", "baz1", 1))
	secondStore.EndWriting()

	thirdStore := SliceStore{}
	thirdStore.BeginWriting()
	thirdStore.WriteRecord(makeLevelDbRecord("a", "foo2", 2))
	thirdStore.WriteRecord(makeLevelDbRecord("b", "bar2", 2))
	thirdStore.WriteRecord(makeLevelDbRecord("c", "baz2", 2))
	thirdStore.EndWriting()

	reader := NewDemuxStoreReader(&firstStore, &secondStore, &thirdStore)
	reader.BeginReading()
	for {
		record, err := reader.ReadRecord()
		if err != nil {
			panic(err)
		}
		if record == nil {
			break
		}
		fmt.Printf("%s: %s\n", record.Key, record.Value)
	}
	reader.EndReading()

	// Output:
	// a: foo2
	// b: bar2
	// c: baz2
	// d: foo0
	// e: foo1
	// f: bar0
	// g: bar1
	// h: baz0
	// i: baz1
}

func ExampleDemuxInputsSorted_duplicateKeys() {
	firstStore := SliceStore{}
	firstStore.BeginWriting()
	firstStore.WriteRecord(makeLevelDbRecord("a", "foo0", 0))
	firstStore.WriteRecord(makeLevelDbRecord("b", "bar0", 0))
	firstStore.WriteRecord(makeLevelDbRecord("c", "baz0", 0))
	firstStore.EndWriting()

	secondStore := SliceStore{}
	secondStore.BeginWriting()
	secondStore.WriteRecord(makeLevelDbRecord("b", "foo1", 1))
	secondStore.WriteRecord(makeLevelDbRecord("c", "bar1", 1))
	secondStore.EndWriting()

	reader := NewDemuxStoreReader(&firstStore, &secondStore)
	reader.BeginReading()
	for {
		record, err := reader.ReadRecord()
		if err != nil {
			panic(err)
		}
		if record == nil {
			break
		}
		fmt.Printf("%s: %s\n", record.Key, record.Value)
	}
	reader.EndReading()

	// Output:
	// a: foo0
	// b: bar0
	// b: foo1
	// c: baz0
	// c: bar1
}

func ExampleMuxedStoreWriter() {
	records := []*LevelDbRecord{
		makeLevelDbRecord("a", "b", 0),
		makeLevelDbRecord("c", "d", 1),
		makeLevelDbRecord("e", "f", 0),
		makeLevelDbRecord("g", "h", 1),
	}

	firstStore := SliceStore{}
	secondStore := SliceStore{}

	writer := NewMuxedStoreWriter(&firstStore, &secondStore)
	for _, record := range records {
		if err := writer.WriteRecord(record); err != nil {
			panic(err)
		}
	}

	firstStore.BeginReading()
	for {
		record, err := firstStore.ReadRecord()
		if err != nil {
			panic(err)
		}
		if record == nil {
			break
		}
		fmt.Printf("[0] %s: %s\n", record.Key, record.Value)
	}
	firstStore.EndReading()

	secondStore.BeginReading()
	for {
		record, err := secondStore.ReadRecord()
		if err != nil {
			panic(err)
		}
		if record == nil {
			break
		}
		fmt.Printf("[1] %s: %s\n", record.Key, record.Value)
	}
	secondStore.EndReading()

	// Output:
	// [0] a: b
	// [0] e: f
	// [1] c: d
	// [1] g: h
}

func ExampleStoreWriterTruncate() {
	store := &SliceStore{}
	truncatingStore := TruncateBeforeWriting(store)
	truncatingStore.BeginWriting()
	truncatingStore.WriteRecord(makeLevelDbRecord("b", "x", 0))
	truncatingStore.EndWriting()

	store.BeginReading()
	for {
		record, err := store.ReadRecord()
		if err != nil {
			panic(err)
		}
		if record == nil {
			break
		}
		fmt.Printf("[0] %s: %s\n", record.Key, record.Value)
	}
	store.EndReading()

	truncatingStore.BeginWriting()
	truncatingStore.WriteRecord(makeLevelDbRecord("c", "y", 0))
	truncatingStore.WriteRecord(makeLevelDbRecord("a", "z", 0))
	truncatingStore.EndWriting()

	store.BeginReading()
	for {
		record, err := store.ReadRecord()
		if err != nil {
			panic(err)
		}
		if record == nil {
			break
		}
		fmt.Printf("[1] %s: %s\n", record.Key, record.Value)
	}
	store.EndReading()

	// Output:
	// [0] b: x
	// [1] a: z
	// [1] c: y
}

func ExampleReadExcludingRanges() {
	store := SliceStore{}
	store.BeginWriting()
	store.WriteRecord(makeLevelDbRecord("a", "x", 0))
	store.WriteRecord(makeLevelDbRecord("b", "y", 0))
	store.WriteRecord(makeLevelDbRecord("c", "z", 0))
	store.WriteRecord(makeLevelDbRecord("d", "y", 0))
	store.WriteRecord(makeLevelDbRecord("e", "x", 0))
	store.WriteRecord(makeLevelDbRecord("f", "a", 0))
	store.WriteRecord(makeLevelDbRecord("g", "b", 0))
	store.WriteRecord(makeLevelDbRecord("h", "c", 0))
	store.WriteRecord(makeLevelDbRecord("j", "e", 0))
	store.WriteRecord(makeLevelDbRecord("k", "f", 0))
	store.EndWriting()

	excludedStore := SliceStore{}
	excludedStore.BeginWriting()
	excludedStore.WriteRecord(makeLevelDbRecord("c", "e", 0))
	excludedStore.WriteRecord(makeLevelDbRecord("h", "i", 0))
	excludedStore.EndWriting()

	excludingReader := ReadExcludingRanges(&store, &excludedStore)
	excludingReader.BeginReading()
	for {
		record, err := excludingReader.ReadRecord()
		if err != nil {
			panic(err)
		}
		if record == nil {
			break
		}
		fmt.Printf("%s ", record.Key)
	}
	excludingReader.EndReading()

	// Output:
	// a b f g j k
}

func ExampleSliceStore() {
	store := SliceStore{}
	store.BeginWriting()
	store.WriteRecord(makeLevelDbRecord("b", "x", 0))
	store.WriteRecord(makeLevelDbRecord("c", "y", 0))
	store.WriteRecord(makeLevelDbRecord("a", "z", 0))
	store.EndWriting()

	store.BeginReading()
	for {
		record, err := store.ReadRecord()
		if err != nil {
			panic(err)
		}
		if record == nil {
			break
		}
		fmt.Printf("[0] %s: %s\n", record.Key, record.Value)
	}
	store.EndReading()

	// Output:
	// [0] a: z
	// [0] b: x
	// [0] c: y
}

func ExampleSliceStore_seek() {
	store := SliceStore{}
	store.BeginWriting()
	store.WriteRecord(makeLevelDbRecord("a", "x", 0))
	store.WriteRecord(makeLevelDbRecord("b", "y", 0))
	store.WriteRecord(makeLevelDbRecord("c", "z", 0))
	store.WriteRecord(makeLevelDbRecord("d", "y", 0))
	store.WriteRecord(makeLevelDbRecord("e", "x", 0))
	store.EndWriting()

	store.BeginReading()
	store.Seek([]byte("c"))
	for {
		record, err := store.ReadRecord()
		if err != nil {
			panic(err)
		}
		if record == nil {
			break
		}
		fmt.Printf("%s: %s\n", record.Key, record.Value)
	}
	store.EndReading()

	// Output:
	// c: z
	// d: y
	// e: x
}