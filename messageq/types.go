package messageq

import (
	"github.com/satori/go.uuid"
)

// An arbitrary message object that can be directly used by applications
type ArbitraryMessage map[string]interface{}

func (t ArbitraryMessage) Id() []byte {
	if id, ok := t["id"]; ok {
		return []byte(id.(string))
	}
	id := uuid.NewV4().String()
	t["id"] = id
	return []byte(id)
}

// A struct that implements Ider to be used in message objects for applications.
// Use like so:
//
//    type MyMessage struct {
//      StructuredMessage
//      OtherFields string
//    }
type StructuredMessage struct {
	MqId string `json:"id"`
}

func (t *StructuredMessage) Id() []byte {
	if t == nil {
		t = new(StructuredMessage)
	}

	if t.MqId == "" {
		t.MqId = uuid.NewV4().String()
	}
	return []byte(t.MqId)
}
