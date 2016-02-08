// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

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
func (w *headerWriter) Write(p []byte) (int, error) {
	// TODO: logic for wrapping headers is actually pretty complex for some header types, like received headers
	var total int
	for len(p)+w.curLineLen > w.maxLineLen {
		toWrite := w.maxLineLen - w.curLineLen
		// Wrap at last space, if any
		lastSpace := bytes.LastIndexByte(p[:toWrite], byte(' '))
		if lastSpace > 0 {
			toWrite = lastSpace
		}
		written, err := w.w.Write(p[:toWrite])
		total += written
		if err != nil {
			return total, err
		}
		written, err = w.w.Write([]byte("\r\n "))
		total += written
		if err != nil {
			return total, err
		}
		p = p[toWrite:]
		w.curLineLen = 1 // Continuation lines are indented
	}
	written, err := w.w.Write(p)
	total += written
	w.curLineLen += written
	return total, err
}

// base64Writer ...
type base64Writer struct {
	w          io.Writer
	curLineLen int
	maxLineLen int
}

// Write ...
func (w *base64Writer) Write(p []byte) (int, error) {
	var total int
	for len(p)+w.curLineLen > w.maxLineLen {
		toWrite := w.maxLineLen - w.curLineLen
		written, err := w.w.Write(p[:toWrite])
		total += written
		if err != nil {
			return total, err
		}
		written, err = w.w.Write([]byte("\r\n"))
		total += written
		if err != nil {
			return total, err
		}
		p = p[toWrite:]
		w.curLineLen = 0
	}
	written, err := w.w.Write(p)
	total += written
	w.curLineLen += written
	return total, err
}
