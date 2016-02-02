package email

// IsFeedbackReportMessage ...
func (m *Message) IsFeedbackReportMessage() bool {
	return m.MessageMedia == "message/feedback-report"
}
