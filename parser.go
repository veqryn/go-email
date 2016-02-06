package email

import (
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
)

// NewMessage ...
func NewMessage(r io.Reader) (*Message, error) {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return nil, err
	}
	return NewMessageWithHeader(Header(msg.Header), msg.Body)
}

// NewMessageWithHeader ...
func NewMessageWithHeader(headers Header, bodyReader io.Reader) (*Message, error) {

	var err error
	var contentType string
	var boundary string
	var subMessage *Message
	contentTypeParams := make(map[string]string)
	body := make([]byte, 0, 0)
	parts := make(map[string]*Message)

	if contentType := headers.Get("Content-Type"); len(contentType) > 0 {
		contentType, contentTypeParams, err = mime.ParseMediaType(contentType)
		if err != nil {
			return nil, err
		}
	}

	contentType = strings.ToLower(contentType)
	if strings.HasPrefix(contentType, "multipart") {
		boundary = contentTypeParams["boundary"]
	}

	// Can only have one of the following: Parts, SubMessage, or Body
	if len(boundary) > 0 {
		err = readParts(contentType, contentTypeParams, bodyReader, boundary, parts)

	} else if strings.HasPrefix(contentType, "message") {
		subMessage, err = NewMessage(bodyReader)

	} else {
		body, err = ioutil.ReadAll(bodyReader)
	}
	if err != nil {
		return nil, err
	}

	return &Message{
		Header:     headers,
		Body:       body,
		SubMessage: subMessage,
		Parts:      parts,
	}, nil
}

// readParts ...
func readParts(messageMedia string, messageMediaParams map[string]string, bodyReader io.Reader, boundary string, parts map[string]*Message) error {
	multipartReader := multipart.NewReader(bodyReader, boundary)
	for part, partErr := multipartReader.NextPart(); partErr != io.EOF; part, partErr = multipartReader.NextPart() {
		if partErr != nil && partErr != io.EOF {
			return partErr
		}
		newEmailPart, msgErr := NewMessageWithHeader(Header(part.Header), part)
		part.Close()
		if msgErr != nil {
			return msgErr
		}
		newPartContentType, _, contentErr := newEmailPart.Header.ContentType()
		if contentErr != nil {
			return contentErr
		}
		parts[newPartContentType] = newEmailPart
	}
	return nil
}
