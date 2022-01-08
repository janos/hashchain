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
type Writer[T any] struct {
	w            io.ReadWriteSeeker
	hasher       hash.Hash
	hashSize     int
	messageSize  int
	encode       func([]byte, T) (int, error)
	lastRecordID int
	buf          []byte
	mu           sync.Mutex
}

// NewWriter creates a new hashcahin Writer that will append new messages to the
// provider io.ReadWriteSeeker. Integrity checksums will be constructed with the
// hasher. It is required to provide the message encoded size information. All
// written encoded messages have to be of the same size.
func NewWriter[T any](w io.ReadWriteSeeker, newHasher func() hash.Hash, encode func([]byte, T) (int, error), messageSize int) (*Writer[T], error) {
	offset, err := w.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("seek to the end of the chain: %w", err)
	}
	hasher := newHasher()
	hashSize := hasher.Size()
	// create a buffer to store data on every record write to reduce allocations
	buf := make([]byte, hashSize+timestampSize+messageSize+hashSize)
	if offset > int64(hashSize) {
		// read the hash of the last record to the buffer
		o, err := readAt(w, offset-int64(hashSize), buf[:hashSize])
		if err != nil {
			return nil, fmt.Errorf("read last hash: %w", err)
		}
		offset = o
	}
	return &Writer[T]{
		w:            w,
		hasher:       hasher,
		hashSize:     hashSize,
		messageSize:  messageSize,
		encode:       encode,
		lastRecordID: int(offset/int64(timestampSize+messageSize+hashSize)) - 1,
		buf:          buf,
	}, nil
}

// Write appends the timestamp and the message to the hashchain. The encoded
// message size has to be the same as specified to NewWriter or
// ErrIncompleteWrite will be returned. This function returns the ID of the
// written record that can be used to read the message and the hash for
// integrity validation.
func (w *Writer[T]) Write(t time.Time, message T) (id int, hash []byte, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// encode time at the place after the hash of the last record
	binary.BigEndian.PutUint64(w.buf[w.hashSize:w.hashSize+timestampSize], uint64(t.UnixNano()))
	// copy message after the previously stored timestamp

	b := w.buf[w.hashSize+timestampSize : w.hashSize+timestampSize+w.messageSize]

	n, err := w.encode(b, message)
	if err != nil {
		return 0, nil, fmt.Errorf("encode: %w", err)
	}
	if n != w.messageSize {
		return 0, nil, ErrIncompleteWrite
	}

	// calculate the hash of previous record's hash, current record timestamp and
	// message
	w.hasher.Reset()
	w.hasher.Write(w.buf[:w.hashSize+timestampSize+w.messageSize])
	// append the hash of the current record after the message
	w.buf = w.hasher.Sum(w.buf[:w.hashSize+timestampSize+w.messageSize])

	if _, err := w.w.Seek(0, io.SeekEnd); err != nil {
		return 0, nil, fmt.Errorf("seek to the end of the hash chain: %w", err)
	}

	// write the record (excluding previous record hash)
	if _, err := w.w.Write(w.buf[w.hashSize:]); err != nil {
		return 0, nil, fmt.Errorf("write data: %w", err)
	}

	hash = make([]byte, w.hashSize)
	// copy the current record hash to be returned safely
	copy(hash, w.buf[w.hashSize+timestampSize+w.messageSize:])
	// copy the hash to the end of the beginning of the buffer
	// for next write to use it for hashing
	copy(w.buf[:w.hashSize], hash)
	w.lastRecordID++

	return w.lastRecordID, hash, nil
}
