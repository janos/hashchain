// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hashchain_test

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"resenje.org/hashchain"
)

func ExampleMain() {
	// Make a temporary file, just for demonstration.
	// You would like to preserve it in real usage.
	f, err := ioutil.TempFile("", "hashchain-example")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
		if err := os.Remove(f.Name()); err != nil {
			log.Fatal(err)
		}
	}()

	messageLength := 34

	encodeFunc := func(b []byte, s string) (int, error) {
		return copy(b, []byte(s)), nil
	}

	decodeFunc := func(s *string, b []byte) (int, error) {
		*s = string(b)
		return len(*s), nil
	}

	// Create a writer and write some messages.
	writer, err := hashchain.NewWriter(f, sha256.New, encodeFunc, messageLength)
	if err != nil {
		log.Fatal(err)
	}
	_, _, err = writer.Write(time.Now(), "something interesting has happened")
	if err != nil {
		log.Fatal(err)
	}
	id2, _, err := writer.Write(time.Now(), "something else has happened, again")
	if err != nil {
		log.Fatal(err)
	}

	// Create a reader, read one message and iterate on all messages.
	reader := hashchain.NewReader(f, sha256.New, decodeFunc, messageLength)

	r, err := reader.Read(id2)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("got second message:", r.Message)

	if err := reader.Iterate(-1, func(r *hashchain.Record[string]) (bool, error) {
		fmt.Println("iterate on message:", r.ID, r.Message)
		return true, nil
	}); err != nil {
		log.Fatal(err)
	}

	// Output: got second message: something else has happened, again
	// iterate on message: 1 something else has happened, again
	// iterate on message: 0 something interesting has happened
}
