package example

import (
	"fmt"
	"github.com/Rafflecopter/golang-messageq/messageq"
	"time"
)

type ExampleMessage struct {
	messageq.StructuredMessage

	Name           string
	Sent, Received int64
}

func (exmsg *ExampleMessage) String() string {
	return fmt.Sprintf(
		"ExampleMessage %s sent: %s received: %s (duration: %s)",
		exmsg.Name,
		time.Unix(0, exmsg.Sent),
		time.Unix(0, exmsg.Received),
		time.Unix(0, exmsg.Received).Sub(time.Unix(0, exmsg.Sent)))
}
