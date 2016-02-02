package email

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/textproto"
)

// IsDeliveryStatusMessage ...
func (m *Message) IsDeliveryStatusMessage() bool {
	return m.MessageMedia == "message/delivery-status"
}

// DeliveryStatusRecipientDNS ...
func (m *Message) DeliveryStatusRecipientDNS() ([]Header, error) {
	recipientDNS := make([]Header, 0, 1)
	if !m.IsDeliveryStatusMessage() {
		return recipientDNS, errors.New("Message not of media content type message/delivery-status, is type: " + m.MessageMedia)
	}
	var err error
	var recipientHeaders textproto.MIMEHeader
	tp := textproto.NewReader(bufio.NewReader(bytes.NewReader(m.Body)))
	for err != io.EOF {
		recipientHeaders, err = tp.ReadMIMEHeader()
		if err != nil && err != io.EOF {
			return nil, err
		}
		recipientDNS = append(recipientDNS, Header(recipientHeaders))
	}
	return recipientDNS, nil
}
