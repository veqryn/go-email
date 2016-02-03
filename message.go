/*
Package email ...
*/
package email

import (
	"errors"
	"fmt"
	"strings"
)

// Message ...
type Message struct {
	// MessageMedia represent the media type of this Message struct,
	// including the Headers and Body/SubMessage/Parts Content.
	MessageMedia string
	// MessageMediaParams is a map of any parameters for the MessageMedia Content-Type
	// such as boundary="1234abcd" and charset=ISO-8859-1
	MessageMediaParams map[string]string

	// ContentMedia represent the media type of this Message's Content (the Body/SubMessage/Parts).
	ContentMedia string
	// ContentMediaParams is a map of any parameters for the ContentMedia Content-Type
	// such as boundary="1234abcd" and charset=ISO-8859-1
	ContentMediaParams map[string]string

	// Header is this message's key-value MIME-style pairs in its header.
	Header Header

	// Can only have one of the following:

	// Parts is a map of the Content-Type string to the *Message of that type,
	// and this map is full in the case where this Message has a Content-Type of "multipart".
	Parts map[string]*Message
	// SubMessage is an encapsulated message, and is full in the case
	// where this Message has a Content-Type of "message".
	SubMessage *Message
	// Body is a byte array of the body of this message, and is full
	// whenever this message doesn't have a Content-Type of "multipart" or "message".
	Body []byte
}

// HasParts ...
func (m *Message) HasParts() bool {
	return len(m.Parts) > 0
}

// HasSubMessage ...
func (m *Message) HasSubMessage() bool {
	return m.SubMessage != nil
}

// HasBody ...
func (m *Message) HasBody() bool {
	return len(m.Body) > 0
}

// Content will return the content of the message, which can only be one the
// following: Body ([]byte), SubMessage (*Message), or Parts (map[string]*Message)
func (m *Message) Content() interface{} {
	if m.HasParts() {
		return m.Parts
	}
	if m.HasSubMessage() {
		return m.SubMessage
	}
	return m.Body // Could still be empty
}

// TextPlain ...
func (m *Message) TextPlain() (string, error) {
	content, err := m.ContentOfType("text/plain")
	if b, ok := content.([]byte); ok {
		return string(b), err
	}
	return "", fmt.Errorf("Unable to cast %T to []byte of content: %v", content, content)
}

// TextHTML ...
func (m *Message) TextHTML() (string, error) {
	content, err := m.ContentOfType("text/html")
	if b, ok := content.([]byte); ok {
		return string(b), err
	}
	return "", fmt.Errorf("Unable to cast %T to []byte of content: %v", content, content)
}

// FindText ...
func (m *Message) FindText() (map[string]string, error) {
	content, err := m.FindContentOfType("text")
	if err != nil {
		return map[string]string{}, err
	}
	return interfaceToStringMap(content)
}

// BodyOfType ...
func (m *Message) BodyOfType(mediaType string) ([]byte, error) {
	content, err := m.ContentOfType(mediaType)
	if b, ok := content.([]byte); ok {
		return b, err
	}
	return []byte{}, fmt.Errorf("Unable to cast %T to []byte of content: %v", content, content)
}

// ContentOfType ...
func (m *Message) ContentOfType(mediaType string) (interface{}, error) {
	if m.HasParts() {
		if val, ok := m.Parts[mediaType]; ok {
			return val.Content(), nil
		}
		return nil, errors.New("Missing Media Type: " + mediaType)
	}
	if m.HasSubMessage() {
		return m.SubMessage.ContentOfType(mediaType)
	}
	if m.ContentMedia == mediaType {
		return m.Body, nil
	}
	return nil, errors.New("Missing Media Type: " + mediaType)
}

// FindBodyOfType ...
func (m *Message) FindBodyOfType(mediaType string) (map[string][]byte, error) {
	content, err := m.FindContentOfType(mediaType)
	if err != nil {
		return map[string][]byte{}, err
	}
	return interfaceMapToBytesMap(content)
}

// FindContentOfType can be used to search for all of a major content type, such as:
// text, message, image, audio, video, application.
func (m *Message) FindContentOfType(mediaType string) (map[string]interface{}, error) {
	if m.HasParts() {
		contents := make(map[string]interface{})
		for key, val := range m.Parts {
			if strings.Contains(key, mediaType) {
				contents[key] = val.Content()
			}
		}
		if len(contents) > 0 {
			return contents, nil
		}
		return nil, errors.New("Missing Media Type: " + mediaType)
	}
	if m.HasSubMessage() {
		return m.SubMessage.FindContentOfType(mediaType)
	}
	if strings.Contains(m.ContentMedia, mediaType) {
		return map[string]interface{}{m.ContentMedia: m.Body}, nil
	}
	return nil, errors.New("Missing Media Type: " + mediaType)
}
