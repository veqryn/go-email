// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package email

import (
	"io"
	"io/ioutil"
	"mime"
	"path/filepath"
)

// NewMessage will create a multipart email containing plain text, html,
// and optionally attachments (create attachments with NewPartAttachment).
// Example structure with 2 pdf attachments:
//     * multipart/mixed
//     * * multipart/alternative
//     * * * text/plain
//     * * * text/html
//     * * application/pdf (attachment)
//     * * application/pdf (attachment)
func NewMessage(headers Header, textPlain string, html string, attachments ...*Message) *Message {

	headers.Set("Content-Type", "multipart/mixed; boundary=\""+RandomBoundary()+"\"")

	alternativePart := NewPartMultipart("alternative", NewPartText(textPlain), NewPartHTML(html))

	parts := make([]*Message, 0, 1+len(attachments))
	parts = append(parts, alternativePart)
	parts = append(parts, attachments...)
	return &Message{Header: headers, Parts: parts}
}

// NewMessageWithInlines will create a multipart email containing plain text, html,
// and optionally attachments (create attachments with NewPartAttachment),
// where the html part contains inline parts, such as inline images
// (create inline parts with NewPartInline).
// Example structure with 2 inline jpeg's and 2 pdf attachments:
//     * multipart/mixed
//     * * multipart/alternative
//     * * * text/plain
//     * * * multipart/related
//     * * * * text/html
//     * * * * image/jpeg (inline with Content-ID)
//     * * * * image/jpeg (inline with Content-ID)
//     * * application/pdf (attachment)
//     * * application/pdf (attachment)
func NewMessageWithInlines(headers Header, textPlain string, html string, inlines []*Message, attachments ...*Message) *Message {

	headers.Set("Content-Type", "multipart/mixed; boundary=\""+RandomBoundary()+"\"")

	inlineParts := []*Message{NewPartHTML(html)}
	inlineParts = append(inlineParts, inlines...)
	relatedPart := NewPartMultipart("related", inlineParts...)

	alternativePart := NewPartMultipart("alternative", NewPartText(textPlain), relatedPart)

	parts := make([]*Message, 0, 1+len(attachments))
	parts = append(parts, alternativePart)
	parts = append(parts, attachments...)
	return &Message{Header: headers, Parts: parts}
}

// NewPartMultipart will create a multipart part, optionally filled with sub-parts.
// Common values for parameter multipartSubType are: mixed, alternative, related, and report.
// Example: if "mixed" is passed in as multipartSubType, then a "multipart/mixed" part is created.
func NewPartMultipart(multipartSubType string, parts ...*Message) *Message {
	return &Message{
		Header: Header{"Content-Type": []string{"multipart/" + multipartSubType + "; boundary=\"" + RandomBoundary() + "\""}},
		Parts:  parts}
}

// NewPartText creates a "text/plain" part, with the text string as its content
// (do not encode, this will happen automatically when needed).
func NewPartText(textPlain string) *Message {
	return &Message{
		Header: Header{"Content-Type": []string{"text/plain; charset=\"UTF-8\""}},
		Body:   []byte(textPlain)}
}

// NewPartHTML creates a "text/html" part, with the html string as its content
// (do not encode, this will happen automatically when needed).
func NewPartHTML(html string) *Message {
	return &Message{
		Header: Header{"Content-Type": []string{"text/html; charset=\"UTF-8\""}},
		Body:   []byte(html)}
}

// NewPartAttachment creates an attachment part,
// using the filename's mime type, and with the reader's content
// (do not encode, this will happen automatically when needed).
func NewPartAttachment(r io.Reader, filename string) (*Message, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return NewPartAttachmentFromBytes(b, filename), nil
}

// NewPartAttachmentFromBytes creates an attachment part,
// using the filename's mime type, and with the bytes as its content
// (do not encode, this will happen automatically when needed).
func NewPartAttachmentFromBytes(raw []byte, filename string) *Message {
	return NewPartFromBytes(raw, mime.TypeByExtension(filepath.Ext(filename)), "attachment; filename=\""+filename+"\"", "")
}

// NewPartInline creates an inline part,
// using the filename's mime type, specified Content-ID
// (do not wrap with angle brackets), and with the reader's content
// (do not encode, this will happen automatically when needed).
func NewPartInline(r io.Reader, filename string, contentID string) (*Message, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return NewPartInlineFromBytes(b, filename, contentID), nil
}

// NewPartInlineFromBytes creates an inline part,
// using the filename's mime type, specified Content-ID
// (do not wrap with angle brackets), and with the bytes as its content
// (do not encode, this will happen automatically when needed).
func NewPartInlineFromBytes(raw []byte, filename string, contentID string) *Message {
	return NewPartFromBytes(raw, mime.TypeByExtension(filepath.Ext(filename)), "inline; filename=\""+filename+"\"", contentID)
}

// NewPart creates a generic binary part,
// using specified contentType, Content-Disposition, Content-ID
// (do not wrap with angle brackets), and with the reader's content
// (do not encode, this will happen automatically when needed).
func NewPart(r io.Reader, contentType string, contentDisposition string, contentID string) (*Message, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return NewPartFromBytes(b, contentType, contentDisposition, contentID), nil
}

// NewPartFromBytes creates a generic binary part,
// using specified contentType, Content-Disposition, Content-ID
// (do not wrap with angle brackets), and with the bytes as its content
// (do not encode, this will happen automatically when needed).
func NewPartFromBytes(raw []byte, contentType string, contentDisposition string, contentID string) *Message {
	headers := Header{}

	if len(contentType) > 0 {
		headers.Set("Content-Type", contentType)
	} else {
		headers.Set("Content-Type", "application/octet-stream")
	}

	if len(contentDisposition) > 0 {
		headers.Set("Content-Disposition", contentDisposition)
	}

	if len(contentID) > 0 {
		headers.Set("Content-ID", "<"+contentID+">")
	}

	return &Message{Header: headers, Body: raw}
}
