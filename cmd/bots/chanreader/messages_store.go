package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/status-im/status-go/cmd/bots"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type messagesStore struct {
	db *leveldb.DB
}

func NewMessagesStore() *messagesStore {
	cwd, _ := os.Getwd()
	db, err := leveldb.OpenFile(cwd+"/messages_store", nil)
	if err != nil {
		log.Fatal("can't open levelDB file. ERR: %v", err)
	}

	return &messagesStore{db}
}

func (ms *messagesStore) Add(message bots.StatusMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	ms.db.Put([]byte(key(message)), []byte(data), nil)
	return nil
}

func (ms *messagesStore) Messages(channel string) []bots.StatusMessage {
	messages := make([]bots.StatusMessage, 0)
	iter := ms.db.NewIterator(util.BytesPrefix([]byte(channel)), nil)
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

func (ms *messagesStore) Close() {
	ms.db.Close()
}

func prefix(message bots.StatusMessage) string {
	return fmt.Sprintf("%s", message.ChannelName)
}

func key(message bots.StatusMessage) string {
	return fmt.Sprintf("%s-%s", prefix(message), message.ID)
}
