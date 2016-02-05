package email

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"time"
)

// interfaceMapToBytesMap ...
func interfaceMapToBytesMap(m map[string]interface{}) (map[string][]byte, error) {
	s := make(map[string][]byte)
	for k, v := range m {
		if b, ok := v.([]byte); ok {
			s[k] = b
		} else {
			return map[string][]byte{}, fmt.Errorf("Unable to cast %T to []byte in map: %v", v, m)
		}
	}
	return s, nil
}

// interfaceToStringMap ...
func interfaceToStringMap(m map[string]interface{}) (map[string]string, error) {
	s := make(map[string]string)
	for k, v := range m {
		if b, ok := v.([]byte); ok {
			s[k] = string(b)
		} else {
			return map[string]string{}, fmt.Errorf("Unable to cast %T to []byte in map: %v", v, m)
		}
	}
	return s, nil
}

var maxInt64 = big.NewInt(math.MaxInt64)

// genMessageID ...
func genMessageID() (string, error) {
	random, err := rand.Int(rand.Reader, maxInt64)
	if err != nil {
		return "", nil
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
	pid := os.Getpid()
	nanoTime := time.Now().UTC().UnixNano()
	return fmt.Sprintf("<%d.%d.%d@%s>", nanoTime, pid, random, hostname), nil
}

// headerWriter ...
type headerWriter struct {
	w          io.Writer
	curLineLen int
	maxLineLen int
}

// Write ...
func (w *headerWriter) Write(p []byte) (n int, err error) {
	// TODO: logic for wrapping headers is actually pretty complex for some header types, like received headers
	var total int
	for len(p)+w.curLineLen > w.maxLineLen {
		toWrite := w.maxLineLen - w.curLineLen
		lastSpace := bytes.LastIndexByte(p[:toWrite], byte(' '))
		if lastSpace > 0 {
			toWrite = lastSpace
		}
		w.w.Write(p[:toWrite])
		w.w.Write([]byte("\r\n"))
		p = p[toWrite:]
		w.w.Write([]byte(" "))
		total += toWrite
		w.curLineLen = 1
	}
	w.w.Write(p)
	w.curLineLen += len(p)
	return total + len(p), nil
}

// base64Writer ...
type base64Writer struct {
	w          io.Writer
	curLineLen int
	maxLineLen int
}

// Write ...
func (w *base64Writer) Write(p []byte) (n int, err error) {
	var total int
	for len(p)+w.curLineLen > w.maxLineLen {
		toWrite := w.maxLineLen - w.curLineLen
		w.w.Write(p[:toWrite])
		w.w.Write([]byte("\r\n"))
		p = p[toWrite:]
		total += toWrite
		w.curLineLen = 0
	}
	w.w.Write(p)
	w.curLineLen += len(p)
	return total + len(p), nil
}
