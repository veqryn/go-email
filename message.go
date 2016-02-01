package email

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
)

// Message ...
type Message struct {
	MediaType         string
	ContentTypeParams map[string]string
	Header            mail.Header
	Body              []byte
	SubMessage        *Message
	Parts             map[string]*Message
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

// NewMessage ...
func NewMessage(r io.Reader) (*Message, error) {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return nil, err
	}
	return newMessageWithHeader(msg.Header, msg.Body)
}

// newMessageWithHeader ...
func newMessageWithHeader(headers mail.Header, bodyReader io.Reader) (*Message, error) {

	var err error
	var mediaType string
	var boundary string
	var subMessage *Message
	contentTypeParams := make(map[string]string)
	body := make([]byte, 0, 0)
	parts := make(map[string]*Message)

	if contentType := headers.Get("Content-Type"); len(contentType) > 0 {
		mediaType, contentTypeParams, err = mime.ParseMediaType(contentType)
		if err != nil {
			return nil, err
		}
	}

	mediaType = strings.ToLower(mediaType)
	if strings.HasPrefix(mediaType, "multipart") {
		boundary = contentTypeParams["boundary"]
	}

	if len(boundary) > 0 {
		err = readParts(bodyReader, boundary, parts)

	} else if strings.HasPrefix(mediaType, "message") {
		subMessage, err = NewMessage(bodyReader)

	} else {
		body, err = ioutil.ReadAll(bodyReader)
	}
	if err != nil {
		return nil, err
	}

	return &Message{
		MediaType:         mediaType,
		ContentTypeParams: contentTypeParams,
		Header:            headers,
		Body:              body,
		SubMessage:        subMessage,
		Parts:             parts,
	}, nil
}

// readParts ...
func readParts(bodyReader io.Reader, boundary string, parts map[string]*Message) error {
	multipartReader := multipart.NewReader(bodyReader, boundary)
	for part, partErr := multipartReader.NextPart(); partErr != io.EOF; part, partErr = multipartReader.NextPart() {
		if partErr != nil && partErr != io.EOF {
			return partErr
		}
		newEmailPart, msgErr := newMessageWithHeader(mail.Header(part.Header), part)
		part.Close()
		if msgErr != nil {
			return msgErr
		}
		parts[newEmailPart.MediaType] = newEmailPart
	}
	return nil
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
	if m.MediaType == mediaType {
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
	if strings.Contains(m.MediaType, mediaType) {
		return map[string]interface{}{m.MediaType: m.Body}, nil
	}
	return nil, errors.New("Missing Media Type: " + mediaType)
}

// interfaceMapToBytesMap ...
func interfaceMapToBytesMap(m map[string]interface{}) (map[string][]byte, error) {
	s := make(map[string][]byte)
	for k, v := range m {
		if b, ok := v.([]byte); ok {
			s[k] = b
		} else {
			return map[string][]byte{}, fmt.Errorf("Unable to cast %T to []byte in map: %v", v, m)
		}
	}
	return s, nil
}

// interfaceToStringMap ...
func interfaceToStringMap(m map[string]interface{}) (map[string]string, error) {
	s := make(map[string]string)
	for k, v := range m {
		if b, ok := v.([]byte); ok {
			s[k] = string(b)
		} else {
			return map[string]string{}, fmt.Errorf("Unable to cast %T to []byte in map: %v", v, m)
		}
	}
	return s, nil
}
