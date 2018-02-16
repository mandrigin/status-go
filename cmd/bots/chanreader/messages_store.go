package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/status-im/status-go/cmd/bots"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	maxCount = 10
)

// ByTimestamp implements sort.Interface for []Person based on
// the Timestamp field.
type ByTimestamp []bots.StatusMessage

func (a ByTimestamp) Len() int           { return len(a) }
func (a ByTimestamp) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTimestamp) Less(i, j int) bool { return a[i].Timestamp < a[j].Timestamp }

type messagesStore struct {
	db       *leveldb.DB
	messages []bots.StatusMessage
}

func NewMessagesStore() *messagesStore {
	cwd, _ := os.Getwd()
	db, err := leveldb.OpenFile(cwd+"/messages_store", nil)
	if err != nil {
		log.Fatal("can't open levelDB file. ERR: %v", err)
	}

	// Load data
	messages := messagesFromIterator(db.NewIterator(nil, nil))
	sort.Sort(ByTimestamp(messages))
	log.Println("loaded messages", len(messages), "\n\nV", messages)

	return &messagesStore{db, messages}
}

func (ms *messagesStore) Add(message bots.StatusMessage) error {
	if len(ms.messages) >= maxCount {
		if message.Timestamp < ms.messages[0].Timestamp {
			log.Println("Message is too old, ignoring", message.ID, message.Timestamp)
			return nil
		}
	}
	_, err := ms.db.Get([]byte(key(message)), nil)
	if err == nil {
		log.Println("Message already exists, ignoring", message.ID, message.Timestamp)
	}

	toRemove := make([]bots.StatusMessage, 0)
	fromIdx := 0
	if len(ms.messages) >= maxCount {
		fromIdx = len(ms.messages) - maxCount + 1
		toRemove = make([]bots.StatusMessage, fromIdx)
		copy(toRemove, ms.messages[:fromIdx])
	}
	messages := append(ms.messages[fromIdx:], message)
	sort.Sort(ByTimestamp(messages))
	ms.messages = messages

	return ms.persist(toRemove, message)
}

func (ms *messagesStore) Messages(channel string) []bots.StatusMessage {
	return messagesFromIterator(ms.db.NewIterator(util.BytesPrefix([]byte(channel)), nil))
}

func (ms *messagesStore) Close() {
	ms.db.Close()
}

func (ms *messagesStore) persist(toRemove []bots.StatusMessage, toAdd bots.StatusMessage) error {
	batch := new(leveldb.Batch)

	data, err := json.Marshal(toAdd)
	if err != nil {
		return err
	}
	log.Println("Adding: ", toAdd.ID, toAdd.Timestamp, toAdd.ChannelName)
	batch.Put([]byte(key(toAdd)), []byte(data))

	for _, messageToRemove := range toRemove {
		log.Println("Garbage-colleting: ", messageToRemove.ID, messageToRemove.Timestamp, messageToRemove.ChannelName)
		batch.Delete([]byte(key(messageToRemove)))
	}
	err = ms.db.Write(batch, nil)

	messages := messagesFromIterator(ms.db.NewIterator(nil, nil))
	log.Println("new len (db):", len(messages))

	return err
}

/* Helper functions */

func messagesFromIterator(iter iterator.Iterator) []bots.StatusMessage {
	messages := make([]bots.StatusMessage, 0)

	for iter.Next() {
		// Use key/value.
		var message bots.StatusMessage
		if err := json.Unmarshal(iter.Value(), &message); err != nil {
			log.Println("Cound not unmarshal JSON. ERR: %v", err)
		} else {
			messages = append(messages, message)
		}
	}
	iter.Release()

	return messages
}

func prefix(message bots.StatusMessage) string {
	return fmt.Sprintf("%s", message.ChannelName)
}

func key(message bots.StatusMessage) string {
	return fmt.Sprintf("%s-%s", prefix(message), message.ID)
}
