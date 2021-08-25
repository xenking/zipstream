# ZipStream

[![Build Status](https://app.travis-ci.com/xenking/zipstream.svg?branch=master)](https://app.travis-ci.com/xenking/zipstream)
[![Go Reference](https://pkg.go.dev/badge/github.com/xenking/zipstream.svg)](https://pkg.go.dev/github.com/xenking/zipstream)
[![Go Report Card](https://goreportcard.com/badge/github.com/xenking/zipstream)](https://goreportcard.com/report/github.com/xenking/zipstream)
Enables zip file streaming from an io.Reader.
Now with ZIP64 support.

## Example

```go
package main

import (
	"github.com/xenking/zipstream"
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
