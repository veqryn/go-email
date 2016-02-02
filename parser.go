package email

import (
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"strings"
)

// NewMessage ...
func NewMessage(r io.Reader) (*Message, error) {
	return NewMessageOfType("message/rfc822", map[string]string{}, r)
}

// NewMessageOfType ...
func NewMessageOfType(messageMedia string, messageMediaParams map[string]string, r io.Reader) (*Message, error) {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return nil, err
	}
	return NewMessageWithHeader(messageMedia, messageMediaParams, textproto.MIMEHeader(msg.Header), msg.Body)
}

// NewMessageWithHeader ...
func NewMessageWithHeader(messageMedia string, messageMediaParams map[string]string, headers textproto.MIMEHeader, bodyReader io.Reader) (*Message, error) {

	var err error
	var contentMediaType string
	var boundary string
	var subMessage *Message
	contentMediaTypeParams := make(map[string]string)
	body := make([]byte, 0, 0)
	parts := make(map[string]*Message)

	if contentType := headers.Get("Content-Type"); len(contentType) > 0 {
		contentMediaType, contentMediaTypeParams, err = mime.ParseMediaType(contentType)
		if err != nil {
			return nil, err
		}
	}

	contentMediaType = strings.ToLower(contentMediaType)
	if strings.HasPrefix(contentMediaType, "multipart") {
		boundary = contentMediaTypeParams["boundary"]
	}

	if len(boundary) > 0 {
		err = readParts(contentMediaType, contentMediaTypeParams, bodyReader, boundary, parts)

	} else if strings.HasPrefix(contentMediaType, "message") {
		subMessage, err = NewMessageOfType(contentMediaType, contentMediaTypeParams, bodyReader)

	} else {
		body, err = ioutil.ReadAll(bodyReader)
	}
	if err != nil {
		return nil, err
	}

	return &Message{
		MessageMedia:       messageMedia,
		MessageMediaParams: messageMediaParams,
		ContentMedia:       contentMediaType,
		ContentMediaParams: contentMediaTypeParams,
		Header:             Header(headers),
		Body:               body,
		SubMessage:         subMessage,
		Parts:              parts,
	}, nil
}

// readParts ...
func readParts(messageMedia string, messageMediaParams map[string]string, bodyReader io.Reader, boundary string, parts map[string]*Message) error {
	multipartReader := multipart.NewReader(bodyReader, boundary)
	for part, partErr := multipartReader.NextPart(); partErr != io.EOF; part, partErr = multipartReader.NextPart() {
		if partErr != nil && partErr != io.EOF {
			return partErr
		}
		newEmailPart, msgErr := NewMessageWithHeader(messageMedia, messageMediaParams, part.Header, part)
		part.Close()
		if msgErr != nil {
			return msgErr
		}
		parts[newEmailPart.ContentMedia] = newEmailPart
	}
	return nil
}
