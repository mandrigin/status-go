package bots

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

type StatusMessage struct {
	ID        string
	From      string
	Text      string
	Timestamp int64
	Raw       string
}

func MessageFromPayload(payload string) StatusMessage {
	message := unrawrChatMessage(payload)
	return StatusMessage{
		Raw: message,
	}
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
	payload := rawrChatMessage(makeChatMessage(text, ch.ChannelName))
	cmd = fmt.Sprintf(cmd, ch.AccountAddress, payload, ch.ChannelKey)
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

func rawrChatMessage(raw string) string {
	bytes := []byte(raw)
	return fmt.Sprintf("0x%s", hex.EncodeToString(bytes))
}

func unrawrChatMessage(message string) string {
	bytes, err := hex.DecodeString(message[2:])
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}

// newUUID generates a random UUID according to RFC 4122
func newUUID() string {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		panic(err)
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

func makeChatMessage(msg string, chat string) string {
	format := `{:message-id "%s", :group-id "%s", :content "%s", :username "Robotic Jet Gopher", :type :public-group-message, :show? true, :clock-value 1, :requires-ack? false, :content-type "text/plain", :timestamp %d}`

	messageID := newUUID()
	timestamp := time.Now().Unix() * 1000

	return fmt.Sprintf(format, messageID, chat, msg, timestamp)
}
