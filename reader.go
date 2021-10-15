// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hashchain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
	"sync"
	"time"
)

// Reader reads records from the hashchain.
type Reader struct {
	r          io.ReadSeeker
	hashSize   int
	recordSize int
	hasherPool *sync.Pool
}

// NewReader creates a new hashchain Reader. It verifies the integrity of the
// hahschain using the provided hasher and it needs a message size in order to
// read records correctly.
func NewReader(r io.ReadSeeker, newHasher func() hash.Hash, messageSize int) *Reader {
	hashSize := newHasher().Size()
	hasherPool := &sync.Pool{
		New: func() interface{} {
			return newHasher()
		},
	}
	return &Reader{
		r:          r,
		hasherPool: hasherPool,
		hashSize:   hashSize,
		recordSize: timestmpSize + messageSize + hashSize,
	}
}

// Read reads the hashchain Record with the provided ID. If the value of the id
// is negative, the last Record will be returned.
func (r *Reader) Read(id int) (*Record, error) {

	if id < 0 {
		offset, err := r.r.Seek(0, io.SeekEnd)
		if err != nil {
			return nil, fmt.Errorf("see to the end of the hash chain: %w", err)
		}
		if offset < int64(r.recordSize) {
			return nil, ErrNotFound
		}
		id = int(offset/int64(r.recordSize)) - 1
	}

	// store the complete record and the hash of the previous one for integrity check
	data := make([]byte, r.hashSize+r.recordSize)
	if id == 0 {
		// read first record without the hash part as there is no previous record
		// leaving the hash part with all zeros
		if _, err := readAt(r.r, 0, data[r.hashSize:]); err != nil {
			if errors.Is(err, io.EOF) {
				return nil, ErrNotFound
			}
			return nil, err
		}
	} else {
		// read the current record completely and the hash of the previous record
		if _, err := readAt(r.r, int64(id*r.recordSize-r.hashSize), data); err != nil {
			if errors.Is(err, io.EOF) {
				return nil, ErrNotFound
			}
			return nil, err
		}
	}

	hash := data[r.recordSize : r.recordSize+r.hashSize]

	if !r.validateIntegrity(hash, data[:r.recordSize]) {
		return nil, ErrIntegrity
	}

	record := &Record{
		ID:   id,
		Hash: hash,
	}
	decodeRecord(data[r.hashSize:r.recordSize], record)

	return record, nil
}

// Iterate reads messages in reverse order as they were written from the start
// ID. If the start ID is negative number, the iteration will start from the
// last record. Message and Hash byte slices in Record passed to the callback
// function are only valid until the function returns and must not be used
// outside of that function as slice content may change during iteration.
func (r *Reader) Iterate(startID int, f func(*Record) (bool, error)) error {
	var offset int64
	if startID < 0 {
		// start from the last record if startID is negative
		var err error
		offset, err = r.r.Seek(0, io.SeekEnd)
		if err != nil {
			return fmt.Errorf("seek to the end of the hash chain: %w", err)
		}
		if offset < int64(r.recordSize) {
			return nil
		}
	} else {
		// seek to the start record position
		startOffset := int64(startID+1) * int64(r.recordSize)
		var err error
		offset, err = r.r.Seek(startOffset, io.SeekStart)
		if err != nil {
			return fmt.Errorf("see to the end start position: %w", err)
		}
		if offset != startOffset {
			return ErrNotFound
		}
	}

	if offset < int64(r.recordSize) {
		return ErrNotFound
	}
	hash := make([]byte, r.hashSize)
	offset, err := readAt(r.r, offset-int64(r.hashSize), hash)
	if err != nil {
		return err
	}

	nextRecordOffset := offset - int64(r.hashSize) - int64(r.recordSize)
	if nextRecordOffset < 0 {
		nextRecordOffset = 0
	}

	data := make([]byte, r.recordSize)
	for {
		if nextRecordOffset == 0 {
			// read the first record without the hash of the non existing
			// previous record
			offset, err = readAt(r.r, nextRecordOffset, data[r.hashSize:])
			if err != nil {
				return fmt.Errorf("seek to the end of the hash chain: %w", err)
			}
			// zero out the hash of the non existing previous record of the
			// first record
			for i := 0; i < r.hashSize; i++ {
				data[i] = 0
			}
		} else {
			offset, err = readAt(r.r, nextRecordOffset, data)
			if err != nil {
				return fmt.Errorf("seek to the end of the hash chain: %w", err)
			}
		}

		id := offset / int64(r.recordSize)

		record := &Record{
			ID:   int(id),
			Hash: hash,
		}

		if !r.validateIntegrity(hash, data[:r.recordSize]) {
			return fmt.Errorf("record %v: %w", id, ErrIntegrity)
		}

		decodeRecord(data[r.hashSize:], record)

		cont, err := f(record)
		if err != nil {
			return fmt.Errorf("record %v function call: %w", id, err)
		}
		if !cont {
			break
		}
		if id == 0 {
			break
		}

		copy(hash, data[:r.hashSize])

		nextRecordOffset = offset - 2*int64(r.recordSize)

		if nextRecordOffset < 0 {
			nextRecordOffset = 0
		}
	}

	return nil
}

func (r *Reader) validateIntegrity(h []byte, data []byte) bool {
	x := r.hasherPool.Get()
	defer r.hasherPool.Put(x)

	hasher := x.(hash.Hash)
	hasher.Reset()
	hasher.Write(data)
	var computed []byte
	computed = hasher.Sum(computed)
	return bytes.Equal(computed, h)
}

func readAt(r io.ReadSeeker, offset int64, data []byte) (int64, error) {
	c, err := r.Seek(offset, io.SeekStart)
	if err != nil {
		return 0, fmt.Errorf("seek %v: %w", offset, err)
	}
	if c != offset {
		return 0, ErrNotFound
	}

	n, err := r.Read(data)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("read %v: %w", offset, err)
	}
	if n != len(data) {
		return 0, ErrIncompleteRead
	}

	return c + int64(n), nil
}

func decodeRecord(data []byte, r *Record) {
	timestamp := int64(binary.BigEndian.Uint64(data[:timestmpSize]))
	r.Time = time.Unix(timestamp/int64(time.Second), timestamp%int64(time.Second))
	r.Message = data[timestmpSize:]
}
