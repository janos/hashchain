// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hashchain

import (
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"sync"
	"time"
)

// Writer appends new messages to the hashchain with a hash to be able to
// validate the integrity of the log.
type Writer struct {
	w               io.ReadWriteSeeker
	hasher          hash.Hash
	hashSize        int
	messageSize     int
	lastRecordID    int
	lastReecordHash []byte
	mu              sync.Mutex
}

// NewWriter creates a new hashcahin Writer that will append new messages to the
// provider io.ReadWriteSeeker. Integrity checksums will be constructed with the
// hasher. It is required to provide the message size information. All written
// messages have to be of the same size.
func NewWriter(w io.ReadWriteSeeker, newHasher func() hash.Hash, messageSize int) (*Writer, error) {
	offset, err := w.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("seek to the end of the log: %w", err)
	}
	hasher := newHasher()
	hashSize := hasher.Size()
	var hash []byte
	if offset > int64(hashSize) {
		hash = make([]byte, hashSize)
		o, err := readAt(w, offset-int64(hashSize), hash)
		if err != nil {
			return nil, fmt.Errorf("read last hash: %w", err)
		}
		offset = o
	}
	return &Writer{
		w:               w,
		hasher:          hasher,
		hashSize:        hashSize,
		messageSize:     messageSize,
		lastRecordID:    int(offset/int64(timestmpSize+messageSize+hashSize)) - 1,
		lastReecordHash: hash,
	}, nil
}

// Write appends the timestamp and the message to the hashchain. The message
// size has to be the same as specified to NewWriter. This function returns the
// ID of the written record that can be used to read the message and the hash
// for integrity validation.
func (w *Writer) Write(t time.Time, message []byte) (id int, hash []byte, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if l := len(message); l != w.messageSize {
		return 0, nil, fmt.Errorf("%w: message size %v instead %v", ErrInvalidMessageSize, l, w.messageSize)
	}

	data := make([]byte, w.hashSize+timestmpSize+w.messageSize, w.hashSize+timestmpSize+w.messageSize+w.hashSize)
	if w.lastReecordHash != nil {
		copy(data[:w.hashSize], w.lastReecordHash)
	}

	binary.BigEndian.PutUint64(data[w.hashSize:w.hashSize+timestmpSize], uint64(t.UnixNano()))
	copy(data[w.hashSize+timestmpSize:w.hashSize+timestmpSize+w.messageSize], message)

	w.hasher.Reset()
	w.hasher.Write(data[:w.hashSize+timestmpSize+w.messageSize])
	data = w.hasher.Sum(data)

	if _, err := w.w.Write(data[w.hashSize:]); err != nil {
		return 0, nil, fmt.Errorf("write data: %w", err)
	}

	hash = data[w.hashSize+timestmpSize+w.messageSize:]

	w.lastReecordHash = hash
	w.lastRecordID++

	return w.lastRecordID, hash, nil
}
