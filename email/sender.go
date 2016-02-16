// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package email

import (
	"errors"
	"net/mail"
	"net/smtp"
)

// Send this email using the SMTP Address:Port, and optionally any SMTP Auth.
func (m *Message) Send(smtpAddressPort string, auth smtp.Auth) error {

	to := m.Header.To()
	cc := m.Header.Cc()
	bcc := m.Header.Bcc()
	all := make([]string, 0, len(to)+len(cc)+len(bcc))

	all = append(append(append(all, to...), cc...), bcc...)
	for i := 0; i < len(all); i++ {
		address, err := mail.ParseAddress(all[i])
		if err != nil {
			return err
		}
		all[i] = address.Address
	}

	if len(all) == 0 {
		return errors.New("May not send email without a recipient (To, CC, or Bcc)")
	}

	from, err := mail.ParseAddress(m.Header.From())
	if err != nil {
		return err
	}

	if len(from.Address) == 0 {
		return errors.New("May not send email without a From address")
	}

	err = m.Save()
	if err != nil {
		return err
	}

	b, err := m.Bytes()
	if err != nil {
		return err
	}

	return smtp.SendMail(smtpAddressPort, auth, from.Address, all, b)
}
