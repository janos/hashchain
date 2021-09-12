# Hash Chain Go library

[![Go](https://github.com/janos/hashchain/workflows/Go/badge.svg)](https://github.com/janos/hashchain/actions)
[![PkgGoDev](https://pkg.go.dev/badge/resenje.org/hashchain)](https://pkg.go.dev/resenje.org/hashchain)
[![NewReleases](https://newreleases.io/badge.svg)](https://newreleases.io/github/janos/hashchain)

Hash Chain is a Go implementation of a compact append only log structure with integrity validation using cryptographic hash functions.

The implementation goals are compact storage size and relatively fast writes and reads.

## Data structure

Data structure that is written by `Writer` and read by `Reader` is a binary representation of a sequence of records that contain a timestamp, message of the specified size and a hash for integrity validation. The data structure is:

```
   8 bytes    messageSize   hashSize
+-----------+-------------+----------+

+-----------+-------------+----------+
| timestamp |   message   |   hash   |  record id 0
+-----------+-------------+----------+
| timestamp |   message   |   hash   |  ...
+-----------+-------------+----------+
| timestamp |   message   |   hash   |  record id n
+-----------+-------------+----------+
```

Where the `n`-th hash is a product of the hash function applied to the concatenated data of the previous (`n-1`-th) record hash, `n`-th record timestamp and `n`-th record message, basically the exact data of the complete record size (hashSize + 8 bytes + messageSize) before the same hash in the structure. The hash used for calculation of the first record is a constant data of zero hash (zero bytes of the length of hashSize).

## Validation

The integrity validation involves successive hashing of records with previous record's hashes and comparing them to the current record hash value. This method ensures detection of changes after the messages is appended to the hash chain structure.

## License

This application is distributed under the BSD-style license found in the [LICENSE](LICENSE) file.
