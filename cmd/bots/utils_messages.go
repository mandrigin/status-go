package bots

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"time"
)

const (
	messageRegexString = `{:message-id "(?P<ID>.+)",\s+:group-id "(?P<GroupID>.+)",\s+:content "(?P<Content>.+)",\s+:username "(?P<Username>.+)",\s+:type :public-group-message.+:timestamp (?P<Timestamp>\d+)}`
)

var messageRegex *regexp.Regexp = regexp.MustCompile(messageRegexString)

type StatusMessage struct {
	ID          string `json:"id"`
	From        string `json:"from"`
	Text        string `json:"text"`
	ChannelName string `json:"channel"`
	Timestamp   int64  `json:"ts"`
	Raw         string
}

func NewStatusMessage(from, text, channel string) StatusMessage {
	return StatusMessage{
		ID:          newUUID(),
		From:        from,
		Text:        text,
		ChannelName: channel,
		Timestamp:   time.Now().Unix() * 1000,
	}
}

func (m StatusMessage) ToPayload() string {
	payloadFormat := `{:message-id "%s", :group-id "%s", :content "%s", :username "%s", :type :public-group-message, :show? true, :clock-value 1, :requires-ack? false, :content-type "text/plain", :timestamp %d}`
	message := fmt.Sprintf(payloadFormat,
		m.ID,
		m.ChannelName,
		m.Text,
		m.From,
		m.Timestamp)

	return rawrChatMessage(message)
}

func MessageFromPayload(payload string) StatusMessage {
	message := unrawrChatMessage(payload)

	r := messageRegex.FindStringSubmatch(message)

	dict := make(map[string]string)
	for idx, name := range messageRegex.SubexpNames() {
		if len(name) > 0 {
			dict[name] = r[idx]
		}
	}

	timestamp, _ := strconv.Atoi(dict["Timestamp"])

	return StatusMessage{
		ID:          dict["ID"],
		From:        dict["Username"],
		Text:        dict["Content"],
		ChannelName: dict["GroupID"],
		Timestamp:   int64(timestamp),
		Raw:         message,
	}
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
