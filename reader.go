// Package zipstream provides support for reading ZIP archives through an io.Reader.
//
// Zip64 archives are not yet supported.
package zipstream

import (
	"archive/zip"
	"bufio"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"io/ioutil"
	"time"
)

const (
	readAhead  = 28
	maxRead    = 4096
	bufferSize = maxRead + readAhead
)

// A Reader provides sequential access to the contents of a zip archive.
// A zip archive consists of a sequence of files,
// The Next method advances to the next file in the archive (including the first),
// and then it can be treated as an io.Reader to access the file's data.
// The Buffered method recovers any bytes read beyond the end of the zip file,
// necessary if you plan to process anything after it that is not another zip file.
type Reader struct {
	io.Reader
	br *bufio.Reader
}

// NewReader creates a new Reader reading from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{br: bufio.NewReaderSize(r, bufferSize)}
}

// Next advances to the next entry in the zip archive.
//
// io.EOF is returned when the end of the zip file has been reached.
// If Next is called again, it will presume another zip file immediately follows
// and it will advance into it.
func (r *Reader) Next() (*zip.FileHeader, error) {
	if r.Reader != nil {
		if _, err := io.Copy(ioutil.Discard, r.Reader); err != nil {
			return nil, err
		}
	}
	sigBytes, err := r.br.Peek(4)
	if err != nil {
		return nil, err
	}

	switch sig := binary.LittleEndian.Uint32(sigBytes); sig {
	case fileHeaderSignature:
		break
	case directoryHeaderSignature: // Directory appears at end of file so we are finished
		return nil, discardCentralDirectory(r.br)
	default:
		return nil, zip.ErrFormat
	}

	headBuf := make([]byte, fileHeaderLen)
	if _, err := io.ReadFull(r.br, headBuf); err != nil {
		return nil, err
	}
	b := readBuf(headBuf[4:])

	f := &zip.FileHeader{
		ReaderVersion:    b.uint16(),
		Flags:            b.uint16(),
		Method:           b.uint16(),
		ModifiedTime:     b.uint16(),
		ModifiedDate:     b.uint16(),
		CRC32:            b.uint32(),
		CompressedSize:   b.uint32(),
		UncompressedSize: b.uint32(),
	}
	f.CompressedSize64 = uint64(f.CompressedSize)
	f.UncompressedSize64 = uint64(f.UncompressedSize)

	filenameLen := int(b.uint16())
	extraLen := int(b.uint16())
	d := make([]byte, filenameLen+extraLen)
	if _, err := io.ReadFull(r.br, d); err != nil {
		return nil, err
	}
	f.Name = string(d[:filenameLen])
	f.Extra = d[filenameLen:]

	utf8Valid1, utf8Require1 := detectUTF8(f.Name)
	switch {
	case !utf8Valid1:
		// Name and Comment definitely not UTF-8.
		f.NonUTF8 = true
	case !utf8Require1:
		// Name and Comment use only single-byte runes that overlap with UTF-8.
		f.NonUTF8 = false
	default:
		// Might be UTF-8, might be some other encoding; preserve existing flag.
		// Some ZIP writers use UTF-8 encoding without setting the UTF-8 flag.
		// Since it is impossible to always distinguish valid UTF-8 from some
		// other encoding (e.g., GBK or Shift-JIS), we trust the flag.
		f.NonUTF8 = f.Flags&0x800 == 0
	}

	needUSize := f.UncompressedSize == ^uint32(0)
	needCSize := f.CompressedSize == ^uint32(0)

	// Best effort to find what we need.
	// Other zip authors might not even follow the basic format,
	// and we'll just ignore the Extra content in that case.
	var modified time.Time

	msdosModified := msDosTimeToTime(f.ModifiedDate, f.ModifiedTime)
	f.Modified = msdosModified
	if !modified.IsZero() {
		f.Modified = modified.UTC()

		// If legacy MS-DOS timestamps are set, we can use the delta between
		// the legacy and extended versions to estimate timezone offset.
		//
		// A non-UTC timezone is always used (even if offset is zero).
		// Thus, FileHeader.Modified.Location() == time.UTC is useful for
		// determining whether extended timestamps are present.
		// This is necessary for users that need to do additional time
		// calculations when dealing with legacy ZIP formats.
		if f.ModifiedTime != 0 || f.ModifiedDate != 0 {
			f.Modified = modified.In(timeZone(msdosModified.Sub(modified)))
		}
	}

	// Assume that uncompressed size 2³²-1 could plausibly happen in
	// an old zip32 file that was sharding inputs into the largest chunks
	// possible (or is just malicious; search the web for 42.zip).
	// If needUSize is true still, it means we didn't see a zip64 extension.
	// As long as the compressed size is not also 2³²-1 (implausible)
	// and the header is not also 2³²-1 (equally implausible),
	// accept the uncompressed size 2³²-1 as valid.
	// If nothing else, this keeps archive/zip working with 42.zip.
	_ = needUSize

	if needCSize {
		return nil, errors.New("zipstream: not a valid zip file")
	}

	dcomp := decompressor(f.Method)
	if dcomp == nil {
		return nil, zip.ErrAlgorithm
	}

	crc := &crcReader{
		hash: crc32.NewIEEE(),
		crc:  &f.CRC32,
	}
	if f.Flags&0x8 != 0 { // If has dataDescriptor
		crc.Reader = dcomp(&descriptorReader{br: r.br, fileHeader: f})
	} else {
		crc.Reader = dcomp(io.LimitReader(r.br, int64(f.CompressedSize64)))
		crc.crc = &f.CRC32
	}
	r.Reader = crc
	return f, nil
}

// Buffered returns any bytes beyond the end of the zip file that it may have
// read. These are necessary if you plan to process anything after it,
// that isn't another zip file.
func (r *Reader) Buffered() io.Reader { return r.br }
