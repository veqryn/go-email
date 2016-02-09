// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

/*
Package email ...
*/
package email

import (
	"fmt"
	"io"
	"mime/quotedprintable"
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

	// Preamble is any text that appears before the first mime multipart,
	// and may only be full in the case where this Message has a Content-Type of "multipart".
	Preamble []byte

	// Epilogue is any text that appears after the last mime multipart,
	// and may only be full in the case where this Message has a Content-Type of "multipart".
	Epilogue []byte

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
	mediaType, _, err := m.Header.ContentType()
	if err != nil {
		return false
	}
	return strings.HasPrefix(mediaType, "multipart")
}

// HasSubMessage ...
func (m *Message) HasSubMessage() bool {
	mediaType, _, err := m.Header.ContentType()
	if err != nil {
		return false
	}
	return strings.HasPrefix(mediaType, "message")
}

// HasBody ...
func (m *Message) HasBody() bool {
	mediaType, _, err := m.Header.ContentType()
	if err != nil && err != ErrHeadersMissingContentType {
		return false
	}
	return !strings.HasPrefix(mediaType, "multipart") && !strings.HasPrefix(mediaType, "message")
}

// MessagesAll ...
func (m *Message) MessagesAll() []*Message {
	return m.MessagesFilter(func(tested *Message) bool {
		return true
	})
}

// MessagesContentTypePrefix ...
func (m *Message) MessagesContentTypePrefix(contentTypePrefix string) []*Message {
	return m.MessagesFilter(func(tested *Message) bool {
		mediaType, _, err := tested.Header.ContentType()
		if err != nil {
			return false
		}
		return strings.HasPrefix(mediaType, contentTypePrefix)
	})
}

// MessagesFilter ...
func (m *Message) MessagesFilter(filter func(*Message) bool) []*Message {

	messages := make([]*Message, 0, 1)
	if filter(m) {
		messages = append(messages, m)
	}

	if m.HasSubMessage() {
		return append(messages, m.SubMessage.MessagesFilter(filter)...)
	}

	if m.HasParts() {
		for _, part := range m.Parts {
			messages = append(messages, part.MessagesFilter(filter)...)
		}
	}
	return messages
}

// Methods required for sending a message:

// WriteTo ...
func (m *Message) WriteTo(w io.Writer) (int64, error) {

	total, err := m.Header.WriteTo(w)
	if err != nil {
		return total, err
	}

	mediaType, mediaTypeParams, err := m.Header.ContentType()
	if err != nil && err != ErrHeadersMissingContentType {
		return total, err
	}
	hasParts := strings.HasPrefix(mediaType, "multipart")
	hasSubMessage := strings.HasPrefix(mediaType, "message")

	if !hasParts && !hasSubMessage {
		return m.writeBody(w, total)
	}

	written, err := io.WriteString(w, "\r\n")
	total += int64(written)
	if err != nil {
		return total, err
	}

	if hasSubMessage {
		written2, err := m.SubMessage.WriteTo(w)
		return total + written2, err

	}
	// hasParts
	return m.writeParts(w, mediaTypeParams["boundary"], total)
}

// writeParts ...
func (m *Message) writeParts(w io.Writer, boundary string, total int64) (int64, error) {

	written, err := w.Write(m.Preamble)
	total += int64(written)
	if err != nil {
		return total, err
	}

	for _, part := range m.Parts {
		written, err = fmt.Fprintf(w, "\r\n--%s\r\n", boundary)
		total += int64(written)
		if err != nil {
			return total, err
		}
		written2, err2 := part.WriteTo(w)
		total += written2
		if err2 != nil {
			return total, err2
		}
	}

	written, err = fmt.Fprintf(w, "\r\n--%s--\r\n", boundary)
	total += int64(written)
	if err != nil {
		return total, err
	}

	written, err = w.Write(m.Epilogue)
	total += int64(written)
	return total, err
}

// writeBody ...
func (m *Message) writeBody(w io.Writer, total int64) (int64, error) {
	var written int
	var err error

	// Encode as quoted-printable if we have a Content-Type, and we do not have a Content-Transfer-Encoding set
	if len(m.Header.Get("Content-Type")) > 0 && len(m.Header.Get("Content-Transfer-Encoding")) == 0 {
		written, err = io.WriteString(w, "Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		w = quotedprintable.NewWriter(w)
	} else {
		written, err = io.WriteString(w, "\r\n")
	}
	total += int64(written)
	if err != nil {
		return total, err
	}

	written, err = w.Write(m.Body)
	return total + int64(written), err
}
