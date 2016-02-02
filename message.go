package email

import (
	"errors"
	"fmt"
	"strings"
)

// Message ...
type Message struct {
	MessageMedia       string
	MessageMediaParams map[string]string

	ContentMedia       string
	ContentMediaParams map[string]string

	Header Header

	Body       []byte
	SubMessage *Message
	Parts      map[string]*Message
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

// Content ...
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

// FindContentOfType ...
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
