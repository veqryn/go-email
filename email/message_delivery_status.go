package email

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/textproto"
)

// HasDeliveryStatusMessage ...
func (m *Message) HasDeliveryStatusMessage() bool {
	contentType, _, err := m.Header.ContentType()
	if err != nil {
		return false
	}
	return contentType == "message/delivery-status" && m.SubMessage != nil
}

// DeliveryStatusMessageDNS ...
func (m *Message) DeliveryStatusMessageDNS() (Header, error) {
	if !m.HasDeliveryStatusMessage() {
		return Header{}, errors.New("Message does not have media content of type message/delivery-status")
	}
	return m.SubMessage.Header, nil
}

// DeliveryStatusRecipientDNS ...
func (m *Message) DeliveryStatusRecipientDNS() ([]Header, error) {
	recipientDNS := make([]Header, 0, 1)
	if !m.HasDeliveryStatusMessage() {
		return recipientDNS, errors.New("Message does not have media content of type message/delivery-status")
	}
	var err error
	var recipientHeaders textproto.MIMEHeader
	tp := textproto.NewReader(bufio.NewReader(bytes.NewReader(m.SubMessage.Body)))
	for err != io.EOF {
		recipientHeaders, err = tp.ReadMIMEHeader()
		if err != nil && err != io.EOF {
			return nil, err
		}
		recipientDNS = append(recipientDNS, Header(recipientHeaders))
	}
	return recipientDNS, nil
}
