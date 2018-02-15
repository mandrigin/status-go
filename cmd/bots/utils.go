package bots

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/status-im/status-go/geth/api"
	"github.com/syndtr/goleveldb/leveldb"
)

type apiHolder struct {
	api *api.StatusAPI
}

func (a *apiHolder) API() *api.StatusAPI {
	return a.api
}

type StatusChannel struct {
	apiHolder
	ChannelName    string
	UserName       string
	FilterID       string
	AccountAddress string
	ChannelKey     string
}

func (ch *StatusChannel) RepeatEvery(ti time.Duration, f func(ch *StatusChannel)) {
	for {
		f(ch)
		time.Sleep(ti)
	}
}

func (ch *StatusChannel) ReadMessages() (result []StatusMessage) {
	cmd := `{"jsonrpc":"2.0","id":2968,"method":"shh_getFilterMessages","params":["%s"]}`
	f := unmarshalJSON(ch.API().CallRPC(fmt.Sprintf(cmd, ch.FilterID)))
	v := f.(map[string]interface{})["result"]
	switch vv := v.(type) {
	case []interface{}:
		for _, u := range vv {
			payload := u.(map[string]interface{})["payload"]
			message := MessageFromPayload(payload.(string))
			result = append(result, message)
		}
	default:
		log.Println(v, "is of a type I don't know how to handle")
	}
	return result
}

func (ch *StatusChannel) SendMessage(text string) {
	cmd := `{"jsonrpc":"2.0","id":0,"method":"shh_post","params":[{"from":"%s","topic":"0xaabb11ee","payload":"%s","symKeyID":"%s","sym-key-password":"status","ttl":2400,"powTarget":0.001,"powTime":1}]}`

	message := NewStatusMessage(ch.UserName, text, ch.ChannelName)

	cmd = fmt.Sprintf(cmd, ch.AccountAddress, message.ToPayload(), ch.ChannelKey)
	log.Println("-> SENT:", ch.API().CallRPC(cmd))
}

type StatusSession struct {
	apiHolder
	Address string
}

func (s *StatusSession) Join(channelName, username string) *StatusChannel {

	cmd := fmt.Sprintf(`{"jsonrpc":"2.0","id":2950,"method":"shh_generateSymKeyFromPassword","params":["%s"]}`, channelName)

	f := unmarshalJSON(s.API().CallRPC(cmd))

	key := f.(map[string]interface{})["result"]

	cmd = `{"jsonrpc":"2.0","id":2,"method":"shh_newMessageFilter","params":[{"allowP2P":true,"topics":["0xaabb11ee"],"type":"sym","symKeyID":"%s"}]}`

	f = unmarshalJSON(s.API().CallRPC(fmt.Sprintf(cmd, key)))

	filterID := f.(map[string]interface{})["result"]

	return &StatusChannel{
		apiHolder:      apiHolder{s.API()},
		ChannelName:    channelName,
		UserName:       username,
		FilterID:       filterID.(string),
		AccountAddress: s.Address,
		ChannelKey:     key.(string),
	}
}

func SignupOrLogin(api *api.StatusAPI, password string) *StatusSession {
	cwd, _ := os.Getwd()
	db, err := leveldb.OpenFile(cwd+"/data", nil)
	if err != nil {
		log.Fatal("can't open levelDB file. ERR: %v", err)
	}
	defer db.Close()

	accountAddress := getAccountAddress(db)

	if accountAddress == "" {
		address, _, _, err := api.CreateAccount(password)
		if err != nil {
			log.Fatalf("could not create an account. ERR: %v", err)
		}
		saveAccountAddress(address, db)
		accountAddress = address
	}

	err = api.SelectAccount(accountAddress, password)
	log.Println("Logged in as", accountAddress)
	if err != nil {
		log.Fatalf("Failed to select account. ERR: %+v", err)
	}
	log.Println("Selected account succesfully")

	return &StatusSession{
		apiHolder: apiHolder{api},
		Address:   accountAddress,
	}
}

const (
	KEY_ADDRESS = "hnny.address"
)

func getAccountAddress(db *leveldb.DB) string {
	addressBytes, err := db.Get([]byte(KEY_ADDRESS), nil)
	if err != nil {
		log.Printf("Error while getting address: %v", err)
		return ""
	}
	return string(addressBytes)
}

func saveAccountAddress(address string, db *leveldb.DB) {
	db.Put([]byte(KEY_ADDRESS), []byte(address), nil)
}

func unmarshalJSON(j string) interface{} {
	var v interface{}
	json.Unmarshal([]byte(j), &v)
	return v
}
