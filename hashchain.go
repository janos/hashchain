// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package hashchain implements a compact append only log structure with integrity validation using cryptographic hash functions.

package hashchain

import (
	"errors"
	"time"
)

const (
	timestampSize = 8
)

// Record holds information about the written message.
type Record[T any] struct {
	// ID is the serial number of the message.
	ID int
	// Time is the time of the message.
	Time time.Time
	// Message is the actual stored message.
	Message T
	// Hash is the hash that validates the integrity of messages.
	Hash []byte
}

var (
	ErrNotFound           = errors.New("hashchain: not found")
	ErrIntegrity          = errors.New("hashchain: integrity check failed")
	ErrLogNotInitialized  = errors.New("hashchain: log not initialized")
	ErrInvalidMessageSize = errors.New("hashchain: invalid record size")
	ErrIncompleteRead     = errors.New("hashchain: incomplete read")
	ErrIncompleteWrite    = errors.New("hashchain: incomplete write")
)
