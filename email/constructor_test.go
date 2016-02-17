// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package email

import (
	"bytes"
	"encoding/base64"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/mail"
	"reflect"
	"strings"
	"testing"
)

// TestBasicEmailCreation ...
func TestBasicEmailCreation(t *testing.T) {

	expectedHeaders := map[string][]string{
		"Subject": []string{"Test Subject"},
		"From":    []string{"Test Name <test.from@host.com>"},
		"To":      []string{"test.to@host.com"},
	}
	expectedText := "This is a long body string that will require wrapping, and has some unicode that must be encoded,\r\n非常感谢你"
	expectedHTML := "<html><head><meta charset=\"UTF-8\"><style>.blue { color: blue; }</style></head>\r\n" +
		"<body>This is a long body string with some <em>HTML</em> and <span class=blue>CSS</span>\r\n" +
		"that will require wrapping, and has some unicode that must be encoded,</br>非常感谢你</body></html>"

	// Create test message
	msg := NewMessage(NewHeader("Test Name <test.from@host.com>", "Test Subject", "test.to@host.com"),
		expectedText, expectedHTML)

	// confirm headers
	for expectedField, expectedValue := range expectedHeaders {
		if actualValue, ok := msg.Header[expectedField]; !ok || !reflect.DeepEqual(expectedValue, actualValue) {
			t.Fatal("Header does not match expectedHeaders for:", expectedField, expectedValue, actualValue)
		}
	}

	// Expected structure:
	//     * multipart/mixed
	//     * * multipart/alternative
	//     * * * text/plain
	//     * * * text/html

	// confirm msg is empty except a single part
	if !confirmContentType(msg, "Content-Type", "multipart/mixed", map[string]string{"boundary": ""}) ||
		!confirmHasParts(msg, 1, false, false) {
		t.Fatal("Message does not match expected structure")
	}
	// confirm msg's part is empty except two parts
	if !confirmContentType(msg.Parts[0], "Content-Type", "multipart/alternative", map[string]string{"boundary": ""}) ||
		!confirmHasParts(msg.Parts[0], 2, false, false) {
		t.Fatal("Message does not match expected structure")
	}
	// confirm both parts only have a body
	if !confirmContentType(msg.Parts[0].Parts[0], "Content-Type", "text/plain", map[string]string{"charset": "UTF-8"}) ||
		!confirmContentType(msg.Parts[0].Parts[1], "Content-Type", "text/html", map[string]string{"charset": "UTF-8"}) ||
		!confirmHasBody(msg.Parts[0].Parts[0]) || !confirmHasBody(msg.Parts[0].Parts[1]) {
		t.Fatal("Message does not match expected structure")
	}

	// confirm content
	if string(msg.Parts[0].Parts[0].Body) != expectedText ||
		string(msg.Parts[0].Parts[1].Body) != expectedHTML {
		t.Fatal("Message text content does not match expected text")
	}

	rawBytes := testAgainstSelf(t, msg)
	testBasicAgainstStdLib(t, msg, rawBytes)
}

func testAgainstSelf(t *testing.T, msg *Message) []byte {
	// confirm can write out
	rawBytes, err := msg.Bytes()
	if err != nil {
		t.Fatal("Could not write out message:", err)
	}

	// create a second message by parsing our output
	parsedMsg, err := ParseMessage(bytes.NewReader(rawBytes))
	if err != nil {
		t.Fatal("Could not parse in message:", err)
	}

	// confirm they are deeply equal
	if !reflect.DeepEqual(msg, parsedMsg) {
		t.Fatal("Message does not match its parsed counterpart")
	}
	return rawBytes
}

func testBasicAgainstStdLib(t *testing.T, msg *Message, rawBytes []byte) {
	// confirm stdlib can parse it too
	stdmsg, err := mail.ReadMessage(bytes.NewReader(rawBytes))
	if err != nil {
		t.Fatal("Standard Library could not parse message:", err)
	}

	// confirm stdlib headers match our headers
	if !reflect.DeepEqual(map[string][]string(msg.Header), map[string][]string(stdmsg.Header)) {
		t.Fatal("Message does not match its parsed counterpart")
	}

	// confirm subsequent parts match
	mixedReader := multipart.NewReader(stdmsg.Body, boundary(map[string][]string(stdmsg.Header)))
	alternativePart, err := mixedReader.NextPart()
	if err != nil {
		t.Fatal("Couldn't get next part", err)
	}

	// test the multipart/alternative
	testAlternativeMultipartWithStdLib(t, msg.Parts[0], alternativePart)

	// confirm EOF
	if _, err = mixedReader.NextPart(); err != io.EOF {
		t.Fatal("Should be EOF", err)
	}
}

func testAlternativeMultipartWithStdLib(t *testing.T, originalPart *Message, alternativePart *multipart.Part) {
	// confirm stdlib headers match our headers
	if !reflect.DeepEqual(map[string][]string(originalPart.Header), map[string][]string(alternativePart.Header)) {
		t.Fatal("Message does not match its parsed counterpart")
	}

	// multipart/alternative without inlines should have text/plain and text/html parts
	alternativeReader := multipart.NewReader(alternativePart, boundary(map[string][]string(alternativePart.Header)))

	plainPart, err := alternativeReader.NextPart()
	if err != nil {
		t.Fatal("Couldn't get next part", err)
	}
	testBodyPartWithStdLib(t, originalPart.Parts[0], plainPart)

	htmlPart, err := alternativeReader.NextPart()
	if err != nil {
		t.Fatal("Couldn't get next part", err)
	}
	testBodyPartWithStdLib(t, originalPart.Parts[1], htmlPart)

	// confirm EOF and Close
	if _, err = alternativeReader.NextPart(); err != io.EOF || alternativePart.Close() != nil {
		t.Fatal("Should be EOF", err)
	}
}

func testBodyPartWithStdLib(t *testing.T, originalPart *Message, stdlibPart *multipart.Part) {

	// decode base64 if exists
	var stdlibPartBodyReader io.Reader
	if stdlibPart.Header.Get("Content-Transfer-Encoding") == "base64" {
		stdlibPart.Header.Del("Content-Transfer-Encoding")
		stdlibPartBodyReader = base64.NewDecoder(base64.StdEncoding, stdlibPart)
	} else {
		stdlibPartBodyReader = stdlibPart
	}

	// confirm stdlib headers match our headers
	if !reflect.DeepEqual(map[string][]string(originalPart.Header), map[string][]string(stdlibPart.Header)) {
		t.Fatal("Message header does not match its parsed counterpart")
	}

	// read content
	content, err := ioutil.ReadAll(stdlibPartBodyReader)
	if err != nil || stdlibPart.Close() != nil {
		t.Fatal("Couldn't read or close part body", err)
	}

	// confirm content is deeply equal
	if !reflect.DeepEqual(originalPart.Body, content) {
		t.Fatal("Message body does not match its parsed counterpart")
	}
}

func confirmContentType(msg *Message, typeField string, expectedType string, expectedParams map[string]string) bool {
	actualType, actualParams, err := msg.Header.parseMediaType(typeField)
	if err != nil || actualType != expectedType || len(actualParams) != len(expectedParams) {
		return false
	}
	for field, value := range expectedParams {
		if field == "boundary" {
			if len(actualParams[field]) == 0 {
				return false
			}
		} else if actualParams[field] != value {
			return false
		}
	}
	return true
}

func confirmHasBody(msg *Message) bool {
	if !msg.HasBody() || len(msg.Body) == 0 ||
		msg.HasSubMessage() || msg.SubMessage != nil ||
		msg.HasParts() || len(msg.Parts) > 0 ||
		len(msg.Preamble) > 0 || len(msg.Epilogue) > 0 {

		return false
	}
	return true
}

func confirmHasSubMessage(msg *Message) bool {
	if msg.HasBody() || len(msg.Body) > 0 ||
		!msg.HasSubMessage() || msg.SubMessage == nil ||
		msg.HasParts() || len(msg.Parts) > 0 ||
		len(msg.Preamble) > 0 || len(msg.Epilogue) > 0 {

		return false
	}
	return true
}

func confirmHasParts(msg *Message, expectedNumberOfParts int, hasPreamble bool, hasEpilogue bool) bool {
	if msg.HasBody() || len(msg.Body) > 0 ||
		msg.HasSubMessage() || msg.SubMessage != nil ||
		!msg.HasParts() || len(msg.Parts) != expectedNumberOfParts {

		return false
	}
	if (len(msg.Preamble) > 0 != hasPreamble) ||
		(len(msg.Epilogue) > 0 != hasEpilogue) {
		return false
	}
	return true
}

func boundary(header map[string][]string) string {
	contentType, params, err := mime.ParseMediaType(header["Content-Type"][0])
	if err != nil {
		panic("Couldn't parse Content-Type: " + err.Error())
	}
	boundary := params["boundary"]
	if !strings.HasPrefix(contentType, "multipart") || len(boundary) == 0 {
		panic("Boundary missing")
	}
	return boundary
}
