// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package email

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"net/mail"
	"net/textproto"
	"strings"
	"time"
)

const (
	// MaxHeaderLineLength ...
	MaxHeaderLineLength = 78

	// MaxHeaderTotalLength ...
	MaxHeaderTotalLength = 998
)

// Header represents the key-value MIME-style pairs in a mail message header.
// Based on textproto.MIMEHeader and mail.Header.
type Header map[string][]string

// NewHeader returns a Header for the most typical use case:
// a Subject, a From address, and a slice of To addresses
func NewHeader(subject string, from string, to []string) Header {
	headers := Header{}
	headers.SetSubject(subject)
	headers.SetFrom(from)
	headers.SetTo(to...)
	return headers
}

// textproto.MIMEHeader Methods:

// Add adds the key, value pair to the header.
// It appends to any existing values associated with key.
func (h Header) Add(key, value string) {
	key = textproto.CanonicalMIMEHeaderKey(key)
	h[key] = append(h[key], value)
}

// Set sets the header entries associated with key to
// the single element value.  It replaces any existing
// values associated with key.
func (h Header) Set(key, value string) {
	h[textproto.CanonicalMIMEHeaderKey(key)] = []string{value}
}

// Get gets the first value associated with the given key.
// If there are no values associated with the key, Get returns "".
// Get is a convenience method.  For more complex queries,
// access the map directly.
func (h Header) Get(key string) string {
	if h == nil {
		return ""
	}
	v := h[textproto.CanonicalMIMEHeaderKey(key)]
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

// IsSet tests if a key is present in the Header
func (h Header) IsSet(key string) bool {
	if h == nil {
		return false
	}
	_, ok := h[textproto.CanonicalMIMEHeaderKey(key)]
	return ok
}

// Del deletes the values associated with key.
func (h Header) Del(key string) {
	delete(h, textproto.CanonicalMIMEHeaderKey(key))
}

// mail.Header Methods:

// Date parses the Date header field.
func (h Header) Date() (time.Time, error) {
	return mail.Header(h).Date()
}

// AddressList parses the named header field as a list of addresses.
func (h Header) AddressList(key string) ([]*mail.Address, error) {
	return mail.Header(h).AddressList(key)
}

// Methods required for sending a message:

// Save adds headers for the "Message-Id", "Date", and "MIME-Version",
// if missing.  An error is returned if the Message-Id can not be created.
func (h Header) Save() error {
	if len(h.Get("Message-Id")) == 0 {
		id, err := GenMessageID()
		if err != nil {
			return err
		}
		h.Set("Message-Id", "<"+id+">")
	}
	if len(h.Get("Date")) == 0 {
		h.Set("Date", time.Now().Format(time.RFC822))
	}
	h.Set("MIME-Version", "1.0")
	return nil
}

// Bytes returns the bytes representing this header.  It is a convenience
// method that calls WriteTo on a buffer, returning its bytes.
func (h Header) Bytes() ([]byte, error) {
	buffer := &bytes.Buffer{}
	_, err := h.WriteTo(buffer)
	return buffer.Bytes(), err
}

// WriteTo writes this header out, including every field except for Bcc.
func (h Header) WriteTo(w io.Writer) (int64, error) {
	// TODO: Fix up the header writer, then switch to MaxHeaderLineLength
	writer := &headerWriter{w: w, maxLineLen: MaxHeaderTotalLength}
	var total int64
	// TODO: sort fields (and sort received headers by date)
	for field, values := range h {
		if field == "Bcc" {
			continue // skip writting out Bcc
		}
		for _, val := range values {
			val = textproto.TrimString(val)
			writer.curLineLen = 0 // Reset for next header
			for _, s := range []string{field, ": ", mime.QEncoding.Encode("UTF-8", val), "\r\n"} {
				written, err := io.WriteString(writer, s)
				if err != nil {
					return total, err
				}
				total += int64(written)
			}
		}
	}
	return total, nil
}

// Convenience Methods:

// ContentType parses and returns the media type, any parameters on it,
// and an error if there is no content type header field.
func (h Header) ContentType() (string, map[string]string, error) {
	if contentType := h.Get("Content-Type"); len(contentType) > 0 {
		mediaType, mediaTypeParams, err := mime.ParseMediaType(contentType)
		if err != nil {
			return "", map[string]string{}, err
		}
		return mediaType, mediaTypeParams, nil
	}
	return "", map[string]string{}, ErrHeadersMissingContentType
}

// ErrHeadersMissingContentType ...
var ErrHeadersMissingContentType = errors.New("Message missing header field: Content-Type")

// From ...
func (h Header) From() string {
	return h.Get("From")
}

// SetFrom ...
func (h Header) SetFrom(email string) {
	h.Set("From", email)
}

// To ...
func (h Header) To() []string {
	return strings.Split(h.Get("To"), ", ")
}

// SetTo ...
func (h Header) SetTo(emails ...string) {
	h.Set("To", strings.Join(emails, ", "))
}

// Cc ...
func (h Header) Cc() []string {
	return strings.Split(h.Get("Cc"), ", ")
}

// SetCc ...
func (h Header) SetCc(emails ...string) {
	h.Set("Cc", strings.Join(emails, ", "))
}

// Bcc ...
func (h Header) Bcc() []string {
	return strings.Split(h.Get("Bcc"), ", ")
}

// SetBcc ...
func (h Header) SetBcc(emails ...string) {
	h.Set("Bcc", strings.Join(emails, ", "))
}

// Subject ...
func (h Header) Subject() string {
	return h.Get("Subject")
}

// SetSubject ...
func (h Header) SetSubject(subject string) {
	h.Set("Subject", subject)
}
