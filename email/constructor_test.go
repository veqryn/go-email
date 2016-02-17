// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package email

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/hex"
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
	t.Parallel()

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
	if !confirmValidHeader(msg.Header) {
		t.Fatal("Invalid Message Header")
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
	testMultipartAlternativeStructure(t, msg.Parts[0])

	// confirm content
	if string(msg.Parts[0].Parts[0].Body) != expectedText ||
		string(msg.Parts[0].Parts[1].Body) != expectedHTML {
		t.Fatal("Message text content does not match expected text")
	}

	rawBytes := testMessageAgainstSelf(t, msg)
	testBasicAgainstStdLib(t, msg, rawBytes)
}

func testMultipartAlternativeStructure(t *testing.T, part *Message) {

	// confirm msg's part is empty except two parts
	if !confirmContentType(part, "Content-Type", "multipart/alternative", map[string]string{"boundary": ""}) ||
		!confirmHasParts(part, 2, false, false) {
		t.Fatal("Message does not match expected structure")
	}
	// confirm both parts only have a body
	if !confirmContentType(part.Parts[0], "Content-Type", "text/plain", map[string]string{"charset": "UTF-8"}) ||
		!confirmContentType(part.Parts[1], "Content-Type", "text/html", map[string]string{"charset": "UTF-8"}) ||
		!confirmHasBody(part.Parts[0]) || !confirmHasBody(part.Parts[1]) {
		t.Fatal("Message does not match expected structure")
	}
}

func testBasicAgainstStdLib(t *testing.T, msg *Message, rawBytes []byte) {
	// confirm stdlib can parse it too
	stdlibMsg, err := mail.ReadMessage(bytes.NewReader(rawBytes))
	if err != nil {
		t.Fatal("Standard Library could not parse message:", err)
	}

	// confirm stdlib headers match our headers
	if !reflect.DeepEqual(map[string][]string(msg.Header), map[string][]string(stdlibMsg.Header)) {
		t.Fatal("Message does not match its parsed counterpart")
	}

	// confirm subsequent parts match
	mixedReader := multipart.NewReader(stdlibMsg.Body, boundary(map[string][]string(stdlibMsg.Header)))
	alternativePart, err := mixedReader.NextPart()
	if err != nil {
		t.Fatal("Couldn't get next part", err)
	}

	// test the multipart/alternative
	testMultipartAlternativeWithStdLib(t, msg.Parts[0], alternativePart)

	// confirm EOF
	if _, err = mixedReader.NextPart(); err != io.EOF {
		t.Fatal("Should be EOF", err)
	}
}

func testMultipartAlternativeWithStdLib(t *testing.T, originalPart *Message, stdlibAltPart *multipart.Part) {
	// confirm stdlib headers match our headers
	if !reflect.DeepEqual(map[string][]string(originalPart.Header), map[string][]string(stdlibAltPart.Header)) {
		t.Fatal("Message does not match its parsed counterpart")
	}

	// multipart/alternative without inlines should have text/plain and text/html parts
	alternativeReader := multipart.NewReader(stdlibAltPart, boundary(map[string][]string(stdlibAltPart.Header)))

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
	if _, err = alternativeReader.NextPart(); err != io.EOF || stdlibAltPart.Close() != nil {
		t.Fatal("Should be EOF", err)
	}
}

// TestComplexEmailCreation ...
func TestInlineEmailCreation(t *testing.T) {
	t.Parallel()

	expectedHeaders := map[string][]string{
		"Subject": []string{"Test Subject with unicode 非常感谢你"},
		"From":    []string{"test.from@host.com"},
		"To":      []string{"test.to@host.com, Another To TestName <another.to@host.com>, third.test.to@host.org"},
		"Cc":      []string{"CC TestName <test.cc@host.com>, another.cc@host.net"},
	}
	expectedText := "This is a long body string that will require wrapping, and has some unicode that must be encoded,\r\n非常感谢你"
	expectedHTML := "<html><head><meta charset=\"UTF-8\"><style>.blue { color: blue; }</style></head>\r\n" +
		"<body>This is a long body string with some <em>HTML</em> and <span class=blue>CSS</span>\r\n" +
		"that will require wrapping, and has some unicode that must be encoded,</br>非常感谢你</body></html>"
	expectedPreamble := "This is a MIME-encapsulated multipart message."
	expectedEpilogue := "This is an epilogue, which while technically valid, should never be used"
	expectedCsv := []byte("foo,bar,\r\nbaz,quux,\r\n,ix,\r\npo,柳条制,wum\r\n")
	expectedGif := bytesOrPanic(hex.DecodeString("47494638396114001600c20000ffffffccffff99999" +
		"933333300000000000000000000000021fe4e546869732061727420697320696e20746865207075626c69632064" +
		"6f6d61696e2e204b6576696e204875676865732c206b6576696e68406569742e636f6d2c2053657074656d62657" +
		"220313939350021f90401000001002c000000001400160000036c38babcf1300c40ab9d23be49baefc0146adce7" +
		"8555068900d81268ba56264c0c77be55c227bca69d654811187bbb9aab6824249544a4e46559f29c53256c289df" +
		"44c3f2e96458c8e8126807add457fd6ecf31a6455b7e7b62dfc0eefefe57e815d034785864638115a5a0f09003b"))
	expectedGifContentID := stringOrPanic(GenContentID("pdf.gif"))
	expectedPng := bytesOrPanic(hex.DecodeString(
		"89504e470d0a1a0a0000000d4948445200000010000000100400000000ff684dbc0000000467414d410000afc8" +
			"37058ae90000003f4944415408d7558cc10d00200c02ddc0613a00fb6f63c200d86adbe8bd2e1018a3983a2c174" +
			"a86100311424f7ef12aa5939e9b88fc01af50ace7aa68efaad8bf2e50231a631a440000000049454e44ae426082"))
	expectedPngContentID := stringOrPanic(GenContentID("ascii.png"))

	// Create header
	header := Header{}
	header.SetFrom(expectedHeaders["From"][0])
	header.SetSubject(expectedHeaders["Subject"][0])
	header.SetTo(strings.Split(expectedHeaders["To"][0], ", ")...)
	header.SetCc(strings.Split(expectedHeaders["Cc"][0], ", ")...)

	// Create attachments
	csvAttachment, err := NewPartAttachment(bytes.NewReader(expectedCsv), "wum.txt")
	if err != nil {
		t.Fatal("Unable to create attachment part from csv reader")
	}
	gifInline := NewPartInlineFromBytes(expectedGif, "pdf.gif", expectedGifContentID)
	pngInline, err := NewPartInline(bytes.NewReader(expectedPng), "ascii.png", expectedPngContentID)
	if err != nil {
		t.Fatal("Unable to create inline part from png reader")
	}

	// Create test message
	msg := NewMessageWithInlines(header, expectedText, expectedHTML, []*Message{gifInline, pngInline}, csvAttachment)
	msg.Preamble = []byte(expectedPreamble)
	msg.Epilogue = []byte(expectedEpilogue)

	// confirm headers
	for expectedField, expectedValue := range expectedHeaders {
		if actualValue, ok := msg.Header[expectedField]; !ok || !reflect.DeepEqual(expectedValue, actualValue) {
			t.Fatal("Header does not match expectedHeaders for:", expectedField, expectedValue, actualValue)
		}
	}
	if !confirmValidHeader(msg.Header) {
		t.Fatal("Invalid Message Header")
	}

	// confirm attachment and inline headers
	if ctype, params, err := csvAttachment.Header.ContentType(); err != nil || ctype != "text/plain" || len(params) != 1 || params["charset"] != "utf-8" {
		t.Fatal("Incorrect Content-Type for part")
	}
	if disposition, params, err := csvAttachment.Header.ContentDisposition(); err != nil || disposition != "attachment" || len(params) != 1 || params["filename"] != "wum.txt" {
		t.Fatal("Incorrect Content-Disposition for part")
	}
	if ctype, params, err := gifInline.Header.ContentType(); err != nil || ctype != "image/gif" || len(params) != 0 {
		t.Fatal("Incorrect Content-Type for part")
	}
	if disposition, params, err := gifInline.Header.ContentDisposition(); err != nil || disposition != "inline" || len(params) != 1 || params["filename"] != "pdf.gif" {
		t.Fatal("Incorrect Content-Disposition for part")
	}
	if gifInline.Header.Get("Content-ID") != "<"+expectedGifContentID+">" {
		t.Fatal("Incorrect Content-ID for part")
	}
	if ctype, params, err := pngInline.Header.ContentType(); err != nil || ctype != "image/png" || len(params) != 0 {
		t.Fatal("Incorrect Content-Type for part")
	}
	if disposition, params, err := pngInline.Header.ContentDisposition(); err != nil || disposition != "inline" || len(params) != 1 || params["filename"] != "ascii.png" {
		t.Fatal("Incorrect Content-Disposition for part")
	}
	if pngInline.Header.Get("Content-ID") != "<"+expectedPngContentID+">" {
		t.Fatal("Incorrect Content-ID for part")
	}

	// Expected structure:
	//     * multipart/mixed
	//     * * multipart/alternative
	//     * * * text/plain
	//     * * * multipart/related
	//     * * * * text/html
	//     * * * * image/gif (inline with Content-ID)
	//     * * * * image/png (inline with Content-ID)
	//     * * text/plain (attachment)

	// confirm msg is empty except a single part
	if !confirmContentType(msg, "Content-Type", "multipart/mixed", map[string]string{"boundary": ""}) ||
		!confirmHasParts(msg, 2, true, true) {
		t.Fatal("Message does not match expected structure")
	}
	testMultipartInlineStructure(t, msg.Parts[0])

	// confirm attachments exist
	if !confirmContentType(msg.Parts[1], "Content-Type", "text/plain", map[string]string{"charset": "utf-8"}) || !confirmHasBody(msg.Parts[1]) {
		t.Fatal("Message does not match expected structure")
	}

	// confirm content
	if string(msg.Parts[0].Parts[0].Body) != expectedText ||
		string(msg.Parts[0].Parts[1].Parts[0].Body) != expectedHTML {
		t.Fatal("Message text content does not match expected text")
	}
	if !reflect.DeepEqual(msg.Parts[1].Body, expectedCsv) ||
		!reflect.DeepEqual(msg.Parts[0].Parts[1].Parts[1].Body, expectedGif) ||
		!reflect.DeepEqual(msg.Parts[0].Parts[1].Parts[2].Body, expectedPng) {
		t.Fatal("Message text content does not match expected text")
	}

	rawBytes := testMessageAgainstSelf(t, msg)
	testInlineAgainstStdLib(t, msg, rawBytes)
}

func testMultipartInlineStructure(t *testing.T, part *Message) {

	// confirm msg's part is empty except two parts
	if !confirmContentType(part, "Content-Type", "multipart/alternative", map[string]string{"boundary": ""}) ||
		!confirmHasParts(part, 2, false, false) {
		t.Fatal("Message does not match expected structure")
	}
	// confirm text/plain has body
	if !confirmContentType(part.Parts[0], "Content-Type", "text/plain", map[string]string{"charset": "UTF-8"}) || !confirmHasBody(part.Parts[0]) {
		t.Fatal("Message does not match expected structure")
	}
	testMultipartRelatedStructure(t, part.Parts[1])
}

func testMultipartRelatedStructure(t *testing.T, part *Message) {
	// confirm related part
	if !confirmContentType(part, "Content-Type", "multipart/related", map[string]string{"boundary": ""}) ||
		!confirmHasParts(part, 3, false, false) {
		t.Fatal("Message does not match expected structure")
	}
	// confirm inline parts
	if !confirmContentType(part.Parts[0], "Content-Type", "text/html", map[string]string{"charset": "UTF-8"}) ||
		!confirmContentType(part.Parts[1], "Content-Type", "image/gif", map[string]string{}) ||
		!confirmContentType(part.Parts[2], "Content-Type", "image/png", map[string]string{}) ||
		!confirmHasBody(part.Parts[0]) || !confirmHasBody(part.Parts[1]) || !confirmHasBody(part.Parts[2]) {
		t.Fatal("Message does not match expected structure")
	}
}

func testInlineAgainstStdLib(t *testing.T, msg *Message, rawBytes []byte) {
	// confirm stdlib can parse it too
	stdlibMsg, err := mail.ReadMessage(bytes.NewReader(rawBytes))
	if err != nil {
		t.Fatal("Standard Library could not parse message:", err)
	}

	// confirm stdlib headers match our headers
	// StandardLibrary is not decoding Q-encoded headers. TODO: Re-enable when GoLang does this.
	//if !reflect.DeepEqual(map[string][]string(msg.Header), map[string][]string(stdlibMsg.Header)) {
	//	t.Fatal("Message does not match its parsed counterpart")
	//}

	// confirm subsequent parts match
	mixedReader := multipart.NewReader(stdlibMsg.Body, boundary(map[string][]string(stdlibMsg.Header)))
	alternativePart, err := mixedReader.NextPart()
	if err != nil {
		t.Fatal("Couldn't get next part", err)
	}

	// test the multipart/alternative
	testMultipartInlineWithStdLib(t, msg.Parts[0], alternativePart)

	// test attachments
	attachmentPart, err := mixedReader.NextPart()
	if err != nil {
		t.Fatal("Couldn't get next part", err)
	}
	testBodyPartWithStdLib(t, msg.Parts[1], attachmentPart)

	// confirm EOF
	if _, err = mixedReader.NextPart(); err != io.EOF {
		t.Fatal("Should be EOF", err)
	}
}

func testMultipartInlineWithStdLib(t *testing.T, originalPart *Message, stdlibAltPart *multipart.Part) {
	// confirm stdlib headers match our headers
	if !reflect.DeepEqual(map[string][]string(originalPart.Header), map[string][]string(stdlibAltPart.Header)) {
		t.Fatal("Message does not match its parsed counterpart")
	}

	// multipart/alternative with inlines should have text/plain and multipart/related parts
	alternativeReader := multipart.NewReader(stdlibAltPart, boundary(map[string][]string(stdlibAltPart.Header)))

	plainPart, err := alternativeReader.NextPart()
	if err != nil {
		t.Fatal("Couldn't get next part", err)
	}
	testBodyPartWithStdLib(t, originalPart.Parts[0], plainPart)

	relatedPart, err := alternativeReader.NextPart()
	if err != nil {
		t.Fatal("Couldn't get next part", err)
	}
	testMultipartRelatedWithStdLib(t, originalPart.Parts[1], relatedPart)

	// confirm EOF and Close
	if _, err = alternativeReader.NextPart(); err != io.EOF || stdlibAltPart.Close() != nil {
		t.Fatal("Should be EOF", err)
	}
}

func testMultipartRelatedWithStdLib(t *testing.T, originalPart *Message, stdlibRelatedPart *multipart.Part) {
	// confirm stdlib headers match our headers
	if !reflect.DeepEqual(map[string][]string(originalPart.Header), map[string][]string(stdlibRelatedPart.Header)) {
		t.Fatal("Message does not match its parsed counterpart")
	}

	// multipart/related should have text/html, image/gif, and image/png parts
	relatedReader := multipart.NewReader(stdlibRelatedPart, boundary(map[string][]string(stdlibRelatedPart.Header)))

	htmlPart, err := relatedReader.NextPart()
	if err != nil {
		t.Fatal("Couldn't get next part", err)
	}
	testBodyPartWithStdLib(t, originalPart.Parts[0], htmlPart)

	gifPart, err := relatedReader.NextPart()
	if err != nil {
		t.Fatal("Couldn't get next part", err)
	}
	testBodyPartWithStdLib(t, originalPart.Parts[1], gifPart)

	pngPart, err := relatedReader.NextPart()
	if err != nil {
		t.Fatal("Couldn't get next part", err)
	}
	testBodyPartWithStdLib(t, originalPart.Parts[2], pngPart)

	// confirm EOF and Close
	if _, err = relatedReader.NextPart(); err != io.EOF || stdlibRelatedPart.Close() != nil {
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

func testMessageAgainstSelf(t *testing.T, msg *Message) []byte {
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
		len(msg.Preamble) > 0 || len(msg.Epilogue) > 0 ||
		!reflect.DeepEqual(msg.Payload(), msg.Body) {

		return false
	}
	return true
}

func confirmHasSubMessage(msg *Message) bool {
	if msg.HasBody() || len(msg.Body) > 0 ||
		!msg.HasSubMessage() || msg.SubMessage == nil ||
		msg.HasParts() || len(msg.Parts) > 0 ||
		len(msg.Preamble) > 0 || len(msg.Epilogue) > 0 ||
		!reflect.DeepEqual(msg.Payload(), msg.SubMessage) {

		return false
	}
	return true
}

func confirmHasParts(msg *Message, expectedNumberOfParts int, hasPreamble bool, hasEpilogue bool) bool {
	if msg.HasBody() || len(msg.Body) > 0 ||
		msg.HasSubMessage() || msg.SubMessage != nil ||
		!msg.HasParts() || len(msg.Parts) != expectedNumberOfParts ||
		!reflect.DeepEqual(msg.Payload(), msg.Parts) {

		return false
	}
	if (len(msg.Preamble) > 0 != hasPreamble) ||
		(len(msg.Epilogue) > 0 != hasEpilogue) {
		return false
	}
	return true
}

func confirmValidHeader(h Header) bool {
	h.Save()
	if h.Get("MIME-Version") != "1.0" ||
		len(h.Get("Date")) == 0 ||
		len(h.Get("Message-Id")) == 0 ||
		len(h.Get("Subject")) == 0 ||
		len(h.Get("From")) == 0 ||
		len(h.Get("To")) == 0 ||
		len(h.Get("Content-Type")) == 0 {
		return false
	}
	return true
}

func confirmBodyLineLength(body []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		if len(scanner.Bytes()) > MaxBodyLineLength {
			return false
		}
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

func stringOrPanic(s string, err error) string {
	if err != nil {
		panic(err)
	}
	return s
}

func bytesOrPanic(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}
