# ZipStream

[![Build Status](https://travis-ci.org/gofunky/zipstream.svg)](https://travis-ci.org/gofunky/zipstream)
[![GoDoc](https://godoc.org/github.com/gofunky/zipstream?status.svg)](https://godoc.org/github.com/gofunky/zipstream)
[![Go Report Card](https://goreportcard.com/badge/github.com/gofunky/zipstream)](https://goreportcard.com/report/github.com/gofunky/zipstream)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/2f67bb8354bd4e96941d067ee86fffb7)](https://www.codacy.com/app/gofunky/zipstream?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=gofunky/zipstream&amp;utm_campaign=Badge_Grade)

Enables zip file reading from a io.Reader stream on the fly.

## Example

```go
package main

import (
	"github.com/gofunky/zipstream"
	"bytes"
	"io"
	"log"
	"io/ioutil"
	)

func main() {
	// Read the first compressed file from a zip file.
	var zipFile bytes.Buffer
    zr := zipstream.NewReader(&zipFile)
	meta, err := zr.Next()
	if err != nil {
		if err != io.EOF {
			panic(err)
		}
	}
	log.Printf("file name: %s", meta.Name)
	compressedFile, err := ioutil.ReadAll(zr)
	if err != nil {
		panic(err)
	}
	log.Printf("file content: %s", string(compressedFile[:]))
}
```

## History
https://github.com/golang/go/issues/10568
