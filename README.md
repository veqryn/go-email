# go-email

### Installation
    $ go get github.com/veqryn/go-email/email

### Releases
This is still a work in progress.

### Usage

Parse an existing email:

    import "github.com/veqryn/go-email/email"
    // reader = io.Reader with your raw email text
    // (or use: strings.NewReader(rawString) or bytes.NewReader(rawBytes))
    msg, err := email.ParseMessage(reader)


Find specific part(s) in a "multipart" message:

    for _, part := range msg.PartsContentTypePrefix("text/html") {
        ...
    }


Find specific part(s) within a message and all sub-messages and sub-parts:

    for _, part := range msg.MessagesContentTypePrefix("application/pdf") {
        ...
    }


Walk the full message tree:

    for _, part := range msg.MessagesAll() {
        mediaType, params, err := part.Header.ContentType()
        switch mediaType {
        case "text/plain":
            fmt.Println(part.Body)
        case "application/pdf":
            ...
        }
    }


Get the decoded body of a message or part:

    myBytes := msg.Body


Create a new simple email:

    // text = string with text/plain content, html = string with text/html content
    header := email.NewHeader("This is my subject", "from.address@host.com", []string{"to.address@host.com"})
    msg := email.NewMessage(header, text, html)


Create a new complex email:

    // text = string with text/plain content, html = string with text/html content
    // gopherReader = io.Reader with the content of an image (as an example)
    // docBytes = []byte with the content of a pdf (as an example)

    header := email.Header{}
    h.SetFrom("charlie@gmail.com")
    h.SetTo("john@outlook.com", "laura@yahoo.com")
    h.SetCc("sarah@icloud.com", "chris@gmail.com")
    h.SetSubject("Hello from go-email")

    imagePart, err := email.NewPartAttachment(gopherReader, "gopher.png")
    pdfPart := email.NewPartAttachmentFromBytes(docBytes, "documentation.pdf")

    msg := email.NewMessage(header, text, html, imagePart, pdfPart)


Send an email:

    msg.Send("smtp.gmail.com:587", smtp.PlainAuth("", "username@gmail.com", "1234567890", "smtp.gmail.com"))
