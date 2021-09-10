// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hashchain

import (
	"time"
)

const (
	timestmpSize = 8
)

// Record holds information about the written message.
type Record struct {
	// ID is the serial number of the message.
	ID int
	// Time is the time of the message.
	Time time.Time
	// Message is the actual message data.
	Message []byte
	// Hash is the hash that validates the integrity of messages.
	Hash []byte
}
