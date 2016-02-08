/*
Package email ...
*/
package email

import (
	"strings"
)

const (
	// MaxBodyLineLength ...
	MaxBodyLineLength = 76
)

// Message ...
type Message struct {
	// Header is this message's key-value MIME-style pairs in its header.
	Header Header

	// Can only have one of the following:

	// Parts is a slice of Messages contained within this Message,
	// and is full in the case where this Message has a Content-Type of "multipart".
	Parts []*Message

	// SubMessage is an encapsulated message, and is full in the case
	// where this Message has a Content-Type of "message".
	SubMessage *Message

	// Body is a byte array of the body of this message, and is full
	// whenever this message doesn't have a Content-Type of "multipart" or "message".
	Body []byte
}

/*
Proper construction of a nested multipart message is as follows:
* multipart/mixed
* * multipart/alternative
* * * text/plain
* * * multipart/related
* * * * text/html
* * * * image/jpeg (inline with Content-ID)
* * * * image/jpeg (inline with Content-ID)
* * application/pdf (attachment)
* * application/pdf (attachment)
* * (etc with other attachments...)
With the last listed in any multipart section being the 'preferred' one to show in any client.
Note that having multiple parts with the same Content-Type is legal!
*/

// Payload will return the payload of the message, which can only be one the
// following: Body ([]byte), SubMessage (*Message), or Parts ([]*Message)
func (m *Message) Payload() interface{} {
	if m.HasParts() {
		return m.Parts
	}
	if m.HasSubMessage() {
		return m.SubMessage
	}
	return m.Body
}

// HasParts ...
func (m *Message) HasParts() bool {
	contentType, _, err := m.Header.ContentType()
	if err != nil {
		return false
	}
	return strings.HasPrefix(contentType, "multipart")
}

// HasSubMessage ...
func (m *Message) HasSubMessage() bool {
	contentType, _, err := m.Header.ContentType()
	if err != nil {
		return false
	}
	return strings.HasPrefix(contentType, "message")
}

// HasBody ...
func (m *Message) HasBody() bool {
	contentType, _, err := m.Header.ContentType()
	if err != nil {
		return false
	}
	return !strings.HasPrefix(contentType, "multipart") && !strings.HasPrefix(contentType, "message")
}

// AllMessages ...
func (m *Message) AllMessages() []*Message {

	messages := make([]*Message, 0, 1)
	messages = append(messages, m)

	if m.HasSubMessage() {
		return append(messages, m.SubMessage.AllMessages()...)
	}

	if m.HasParts() {
		for _, part := range m.Parts {
			messages = append(messages, part.AllMessages()...)
		}
	}
	return messages
}
