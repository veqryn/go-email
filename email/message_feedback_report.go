// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package email

// HasFeedbackReportMessage returns true if this Message has a
// content type of "message/feedback-report" and has a non-nil SubMessage.
func (m *Message) HasFeedbackReportMessage() bool {
	contentType, _, err := m.Header.ContentType()
	if err != nil {
		return false
	}
	return contentType == "message/feedback-report" && m.SubMessage != nil
}

// TODO: wip
