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
	MaxHeaderLengthLength = 78
	MaxHeaderTotalLength  = 998
)

// Header represents the key-value MIME-style pairs in a mail message header.
// Based on textproto.MIMEHeader and mail.Header.
type Header map[string][]string

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

// Save ...
func (h Header) Save() error {
	if len(h.Get("Message-Id")) == 0 {
		id, err := genMessageID()
		if err != nil {
			return err
		}
		h.Set("Message-Id", id)
	}
	if len(h.Get("Date")) == 0 {
		h.Set("Date", time.Now().Format(time.RFC822))
	}
	h.Set("MIME-Version", "1.0")
	return nil
}

// WriteTo ...
func (h Header) WriteTo(w io.Writer) (n int64, err error) {
	writer := &headerWriter{w: w, maxLineLen: MaxHeaderLengthLength}
	var total int64
	// TODO: sort fields (and sort received headers by date)
	for field, values := range h {
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

// Bytes ...
func (h Header) Bytes() ([]byte, error) {
	b := bytes.Buffer{}
	_, err := h.WriteTo(&b)
	if err != nil {
		return []byte{}, err
	}
	return b.Bytes(), nil
}

// Convenience Methods:

// ContentType ...
func (h Header) ContentType() (string, map[string]string, error) {
	if contentType := h.Get("Content-Type"); len(contentType) > 0 {
		contentType, contentTypeParams, err := mime.ParseMediaType(contentType)
		if err != nil {
			return "", map[string]string{}, err
		}
		return contentType, contentTypeParams, nil
	}
	return "", map[string]string{}, errors.New("Message missing header field: Content-Type")
}

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
func (h Header) SetTo(emails []string) {
	h.Set("To", strings.Join(emails, ", "))
}

// Cc ...
func (h Header) Cc() []string {
	return strings.Split(h.Get("Cc"), ", ")
}

// SetCc ...
func (h Header) SetCc(emails []string) {
	h.Set("Cc", strings.Join(emails, ", "))
}

// Bcc ...
func (h Header) Bcc() []string {
	return strings.Split(h.Get("Bcc"), ", ")
}

// SetBcc ...
func (h Header) SetBcc(emails []string) {
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
