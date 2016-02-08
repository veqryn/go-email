package email

// HasFeedbackReportMessage ...
func (m *Message) HasFeedbackReportMessage() bool {
	contentType, _, err := m.Header.ContentType()
	if err != nil {
		return false
	}
	return contentType == "message/feedback-report" && m.SubMessage != nil
}

// TODO: wip
