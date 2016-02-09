// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package email

import (
	"bufio"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
)

// NewMessage ...
func NewMessage(r io.Reader) (*Message, error) {
	msg, err := mail.ReadMessage(&leftTrimReader{r: bufio.NewReader(r)})
	if err != nil {
		return nil, err
	}
	return NewMessageWithHeader(Header(msg.Header), msg.Body)
}

// NewMessageWithHeader ...
func NewMessageWithHeader(headers Header, bodyReader io.Reader) (*Message, error) {

	if headers.Get("Content-Transfer-Encoding") == "quoted-printable" {
		headers.Del("Content-Transfer-Encoding")
		bodyReader = quotedprintable.NewReader(bodyReader)
	}

	var err error
	var mediaType string
	var subMessage *Message
	mediaTypeParams := make(map[string]string)
	body := make([]byte, 0, 0)
	parts := make([]*Message, 0, 0)

	if contentType := headers.Get("Content-Type"); len(contentType) > 0 {
		mediaType, mediaTypeParams, err = mime.ParseMediaType(contentType)
		if err != nil {
			return nil, err
		}
	} // Lack of contentType is not a problem

	// Can only have one of the following: Parts, SubMessage, or Body
	if strings.HasPrefix(mediaType, "multipart") {
		parts, err = readParts(mediaType, mediaTypeParams, bodyReader, mediaTypeParams["boundary"])

	} else if strings.HasPrefix(mediaType, "message") {
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
func readParts(messageMedia string, messageMediaParams map[string]string, bodyReader io.Reader, boundary string) ([]*Message, error) {
	// TODO: parse out and save the preamble and epilogue
	parts := make([]*Message, 0, 1)
	multipartReader := multipart.NewReader(bodyReader, boundary)
	for part, partErr := multipartReader.NextPart(); partErr != io.EOF; part, partErr = multipartReader.NextPart() {
		if partErr != nil && partErr != io.EOF {
			return []*Message{}, partErr
		}
		newEmailPart, msgErr := NewMessageWithHeader(Header(part.Header), part)
		part.Close()
		if msgErr != nil {
			return []*Message{}, msgErr
		}
		parts = append(parts, newEmailPart)
	}
	return parts, nil
}
