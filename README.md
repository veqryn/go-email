# go-email
===============

### Installation
    $ go get github.com/veqryn/go-email/email

### Importing
    $ import "github.com/veqryn/go-email/email"

### Releases
This is still a work in progress.

### Usage

Basic:

    import "github.com/veqryn/go-email/email"
    // reader := io.Reader with your raw email text
    msg, err := email.NewMessage(reader)

Walk Message tree:

    for _, part := range msg.MessagesAll() {
        mediaType, params, err := part.Header.ContentType()
        switch mediaType {
        case "text/plain":
            ...
        case "application/pdf":
            ...
        }
    }

Find a specific part or parts in a "multipart" message:

    for _, part := range msg.PartsContentTypePrefix("text/html") {
        ...
    }

Get the decoded body of a message or part:

    myBytes := msg.Body
