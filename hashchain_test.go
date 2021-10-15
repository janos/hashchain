// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hashchain_test

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"resenje.org/hashchain"
)

var (
	hasher = sha256.New
)

func TestHashchain(t *testing.T) {
	f := newFile(t)
	defer f.Close()

	messageSize := 9

	writer, err := hashchain.NewWriter(f, hasher, messageSize)
	assertError(t, err, nil)

	message1 := "message 1"
	message1Time := time.Now()

	id1, hash1, err := writer.Write(message1Time, []byte(message1))
	assertError(t, err, nil)
	if id1 != 0 {
		t.Errorf("got id %v, want 0", id1)
	}

	reader := hashchain.NewReader(f, hasher, messageSize)

	r, err := reader.Read(0)
	assertError(t, err, nil)
	assertRecord(t, r, 0, message1Time, []byte(message1), hash1)

	r, err = reader.Read(-1)
	assertError(t, err, nil)
	assertRecord(t, r, 0, message1Time, []byte(message1), hash1)

	i := 0
	if err := reader.Iterate(-1, func(r *hashchain.Record) (bool, error) {
		switch i {
		case 0:
			assertRecord(t, r, 0, message1Time, []byte(message1), hash1)
		default:
			t.Errorf("got unexpected record %v", r)
		}
		i--
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}

	i = 0
	if err := reader.Iterate(0, func(r *hashchain.Record) (bool, error) {
		switch i {
		case 0:
			assertRecord(t, r, 0, message1Time, []byte(message1), hash1)
		default:
			t.Errorf("got unexpected record %v", r)
		}
		i--
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}

	message2 := "message 2"
	message2Time := time.Now()

	id2, hash2, err := writer.Write(message2Time, []byte(message2))
	assertError(t, err, nil)
	if id2 != 1 {
		t.Errorf("got id %v, want 1", id2)
	}

	if bytes.Equal(hash1, hash2) {
		t.Errorf("hashes are the same: %x and %x", hash1, hash2)
	}

	r, err = reader.Read(1)
	assertError(t, err, nil)
	assertRecord(t, r, 1, message2Time, []byte(message2), hash2)

	r, err = reader.Read(-1)
	assertError(t, err, nil)
	assertRecord(t, r, 1, message2Time, []byte(message2), hash2)

	r, err = reader.Read(0)
	assertError(t, err, nil)
	assertRecord(t, r, 0, message1Time, []byte(message1), hash1)

	i = 1
	if err := reader.Iterate(-1, func(r *hashchain.Record) (bool, error) {
		switch i {
		case 0:
			assertRecord(t, r, 0, message1Time, []byte(message1), hash1)
		case 1:
			assertRecord(t, r, 1, message2Time, []byte(message2), hash2)
		default:
			t.Errorf("got unexpected record %v", r)
		}
		i--
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}

	i = 1
	if err := reader.Iterate(1, func(r *hashchain.Record) (bool, error) {
		switch i {
		case 0:
			assertRecord(t, r, 0, message1Time, []byte(message1), hash1)
		case 1:
			assertRecord(t, r, 1, message2Time, []byte(message2), hash2)
		default:
			t.Errorf("got unexpected record %v", r)
		}
		i--
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}

	i = 0
	if err := reader.Iterate(0, func(r *hashchain.Record) (bool, error) {
		switch i {
		case 0:
			assertRecord(t, r, 0, message1Time, []byte(message1), hash1)
		default:
			t.Errorf("got unexpected record %v", r)
		}
		i--
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestWriterReopen(t *testing.T) {
	f := newFile(t)

	messageSize := 9

	writer, err := hashchain.NewWriter(f, hasher, messageSize)
	assertError(t, err, nil)

	message1 := "message 1"
	message1Time := time.Now()

	id1, hash1, err := writer.Write(message1Time, []byte(message1))
	assertError(t, err, nil)

	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	f, err = os.OpenFile(f.Name(), os.O_RDWR, 0o666)
	assertError(t, err, nil)

	writer, err = hashchain.NewWriter(f, hasher, messageSize)
	assertError(t, err, nil)

	message2 := "message 2"
	message2Time := time.Now()

	id2, hash2, err := writer.Write(message2Time, []byte(message2))
	assertError(t, err, nil)

	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	f, err = os.Open(f.Name())
	assertError(t, err, nil)

	reader := hashchain.NewReader(f, hasher, messageSize)

	i := 1
	if err := reader.Iterate(-1, func(r *hashchain.Record) (bool, error) {
		switch i {
		case 0:
			assertRecord(t, r, id1, message1Time, []byte(message1), hash1)
		case 1:
			assertRecord(t, r, id2, message2Time, []byte(message2), hash2)
		default:
			t.Errorf("got unexpected record %v", r)
		}
		i--
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestReaderNoData(t *testing.T) {
	r := hashchain.NewReader(strings.NewReader(""), hasher, 10)

	for i := -2; i < 2; i++ {
		_, err := r.Read(i)
		assertError(t, err, hashchain.ErrNotFound)

		err = r.Iterate(i, func(*hashchain.Record) (bool, error) {
			return true, nil
		})
		if i < 0 {
			assertError(t, err, nil)
		} else {
			assertError(t, err, hashchain.ErrNotFound)
		}
	}
}

func newFile(t *testing.T) *os.File {
	t.Helper()

	dir := t.TempDir()

	f, err := os.Create(filepath.Join(dir, "hashchain.log"))
	assertError(t, err, nil)

	return f
}

func assertError(t *testing.T, got, want error) {
	t.Helper()

	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

func assertRecord(t *testing.T, got *hashchain.Record, id int, ta time.Time, message, hash []byte) {
	t.Helper()

	if id != got.ID {
		t.Errorf("got id %v, want %v", got.ID, id)
	}
	if !got.Time.Equal(ta) {
		t.Errorf("got time %s, want %s", got.Time, ta)
	}
	if !bytes.Equal(message, got.Message) {
		t.Errorf("got message %x, want %x", got.Message, message)
	}
	if !bytes.Equal(hash, got.Hash) {
		t.Errorf("got hash %x, want %x", got.Hash, hash)
	}
}
