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

// NewMessage will create a multipart email containing plain text, html, and optionally attachments.
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

// NewMessageWithInlines will create a multipart email containing plain text, html, and optionally attachments.
// where the html part contains inline parts (such as inline images).
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

// NewPartMultipart ...
func NewPartMultipart(multipartSubType string, parts ...*Message) *Message {
	return &Message{
		Header: Header{"Content-Type": []string{"multipart/" + multipartSubType + "; boundary=\"" + RandomBoundary() + "\""}},
		Parts:  parts}
}

// NewPartText ...
func NewPartText(textPlain string) *Message {
	return &Message{
		Header: Header{"Content-Type": []string{"text/plain; charset=\"UTF-8\""}},
		Body:   []byte(textPlain)}
}

// NewPartHTML ...
func NewPartHTML(html string) *Message {
	return &Message{
		Header: Header{"Content-Type": []string{"text/html; charset=\"UTF-8\""}},
		Body:   []byte(html)}
}

// NewPartAttachment ...
func NewPartAttachment(r io.Reader, filename string) (*Message, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return NewPartAttachmentFromBytes(b, filename), nil
}

// NewPartAttachmentFromBytes ...
func NewPartAttachmentFromBytes(raw []byte, filename string) *Message {
	return NewPartFromBytes(raw, mime.TypeByExtension(filepath.Ext(filename)), "attachment; filename=\""+filename+"\"", "")
}

// NewPartInline ...
func NewPartInline(r io.Reader, filename string, contentID string) (*Message, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return NewPartInlineFromBytes(b, filename, contentID), nil
}

// NewPartInlineFromBytes ...
func NewPartInlineFromBytes(raw []byte, filename string, contentID string) *Message {
	return NewPartFromBytes(raw, mime.TypeByExtension(filepath.Ext(filename)), "inline; filename=\""+filename+"\"", contentID)
}

// NewPart ...
func NewPart(r io.Reader, contentType string, contentDisposition string, contentID string) (*Message, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return NewPartFromBytes(b, contentType, contentDisposition, contentID), nil
}

// NewPartFromBytes ...
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
