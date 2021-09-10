// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hashchain

import "errors"

var (
	ErrNotFound           = errors.New("hashchain: not found")
	ErrIntegrity          = errors.New("hashchain: integrity check failed")
	ErrLogNotInitialized  = errors.New("hashchain: log not initialized")
	ErrInvalidMessageSize = errors.New("hashchain: invalid record size")

	errIncompleteRead = errors.New("hashchain: incomplete read")
)
