package transformer

import (
	"fmt"

	"github.com/sburnett/lexicographic-tuples"
	"github.com/sburnett/transformer/store"
)

func makeRecord(values ...interface{}) *store.Record {
	return &store.Record{
		Key: lex.EncodeOrDie(values...),
	}
}

func ExampleGrouper() {
	records := make(chan *store.Record, 10)
	records <- makeRecord("hello", int32(10), "foo")
	records <- makeRecord("hello", int32(10), "bar")
	records <- makeRecord("hello", int32(10), "baz")
	records <- makeRecord("hello", int32(20), "foo")
	records <- makeRecord("hello", int32(20), "gorp")
	records <- makeRecord("world", int32(10), "blah")
	records <- makeRecord("whatever", int32(15), "foo")
	close(records)

	var stringKey string
	var intKey int32
	grouper := GroupRecords(records, &stringKey, &intKey)

	for grouper.NextGroup() {
		idx := 0
		for grouper.NextRecord() {
			record := grouper.Read()
			var joinedString string
			lex.DecodeOrDie(record.Key, &joinedString)
			fmt.Printf("[%d] %s %d %s\n", idx, stringKey, intKey, joinedString)
			idx++
		}
	}

	// Output:
	// [0] hello 10 foo
	// [1] hello 10 bar
	// [2] hello 10 baz
	// [0] hello 20 foo
	// [1] hello 20 gorp
	// [0] world 10 blah
	// [0] whatever 15 foo
}
