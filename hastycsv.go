// Package hastycsv is fast, simple, and NOT-RFC-4180-COMPLIANT CSV reader.
//
// Take a look at README and code examples in https://github.com/cet001/hastycsv
// for usage.
package hastycsv

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"unsafe"
)

// Needed by ParseUint32() for better performance.
var base10exp = []uint64{
	1, 10, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000,
	1000000000,
	10000000000,
	100000000000,
	1000000000000,
	10000000000000,
	100000000000000,
	1000000000000000,
	10000000000000000,
	100000000000000000,
	100000000000000000,
	1000000000000000000,
}

// Definition of a callback function that serves as a sequential record iterator.
// Read() and ReadFile() will stop reading the input records if this function
// returns an error.
type Next func(i int, record []Field) error

// Reads records from a CSV-encoded file or io.Reader.
type Reader struct {
	// Comma is the field delimiter.
	// It is set to comma (',') by NewReader.
	// Comma cannot be \r or \n.
	Comma byte

	fields []Field
	row    int
	err    error
}

// Returns a new Reader whose Delimiter is set to the comma character (',').
func NewReader() *Reader {
	return &Reader{
		Comma: ',',
	}
}

func (me *Reader) Read(r io.Reader, nextRecord Next) error {
	if me.Comma == '\r' || me.Comma == '\n' {
		return fmt.Errorf(`Comma delimiter cannot be \r or \n`)
	}

	var fields []Field
	isFirstRecord := true
	delim := me.Comma
	me.row = 0

	lineScanner := bufio.NewScanner(r)
	for lineScanner.Scan() {
		b := lineScanner.Bytes()

		if isFirstRecord {
			// Infer number of fields from the first row and initialize the []fields buffer
			fieldCount := bytes.Count(b, []byte{delim}) + 1

			fields = make([]Field, fieldCount)
			for i := 0; i < fieldCount; i++ {
				field := &fields[i]
				field.reader = me
			}
			isFirstRecord = false
		}

		me.row++

		if err := splitBytes(b, delim, fields); err != nil {
			return fmt.Errorf("Line %v: %v: \"%v\"", me.row, err, string(b))
		}

		callbackErr := nextRecord(me.row, fields)

		if me.err != nil {
			return fmt.Errorf("Line %v: %v", me.row, me.err)
		} else if callbackErr != nil {
			return fmt.Errorf("Line %v: %v", me.row, callbackErr)
		}
	}

	if me.err != nil {
		return fmt.Errorf("Line %v: %v", me.row, me.err)
	}

	if err := lineScanner.Err(); err != nil {
		return fmt.Errorf("Error scanning input: %v", err)
	}

	return nil
}

func ReadFile(csvFilePath string, comma byte, nextRecord Next) error {
	f, err := os.Open(csvFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	r := NewReader()
	r.Comma = comma
	return r.Read(bufio.NewReaderSize(f, 32*1024), nextRecord)
}

// Represents a field (encoded as a UTF-8 string) within a CSV record.
type Field struct {
	reader *Reader
	data   []byte
}

// Returns true if this field is empty.
func (me Field) IsEmpty() bool {
	return len(me.data) == 0
}

// Returns the backing byte slice of this field.
func (me Field) Bytes() []byte {
	return me.data
}

// Returns this field as a string.
func (me Field) String() string {
	return string(me.data)
}

// Interprets this field as an ASCII string and performs an in-place conversion
// to lowercase.
func (me Field) ToLower() Field {
	for i, ch := range me.data {
		if ch >= 'A' && ch <= 'Z' {
			me.data[i] += 32 // make this acii character lowercase (e.g. 'A' => 'a')
		}
	}

	return me
}

// Parses this field as a Uint32.
func (me Field) Uint32() uint32 {
	i, err := ParseUint32(me.data)
	if err != nil {
		if me.reader.err == nil {
			me.reader.err = fmt.Errorf(`Can't parse field as uint32: %v`, err)
		}
	}

	return i
}

// Parses this field as a float32.
func (me Field) Float32() float32 {
	f, err := strconv.ParseFloat(me.unsafeString(), 32)
	if err != nil {
		if me.reader.err == nil {
			me.reader.err = err
		}
		return 0
	}
	return float32(f)
}

// ParseUint32() parses an ascii byte array into a uint32 value.
func ParseUint32(data []byte) (uint32, error) {
	d := len(data)
	if d > 10 { // 2^32 is 10 digits long
		return 0, fmt.Errorf(`"%v" is too long to be parsed as a uint32`, string(data))
	}

	v := uint64(0)
	for _, ch := range data {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf(`"%v" contains non-numeric character '%v'`, string(data), string(ch))
		}
		d--
		v += uint64(ch-'0') * base10exp[d]
	}

	if v > math.MaxUint32 {
		return 0, fmt.Errorf(`"%v" overflows uint32`, string(data))
	}

	return uint32(v), nil
}

// Returns the string representation of this Field without creating a memory allocation.
//
// WARNING! The returned string points to this Field object, which is a mutable
// byte slice!
func (me Field) unsafeString() string {
	return *(*string)(unsafe.Pointer(&me.data))
}

// Analogous to strings.Split(), this function splits a byte slice into a slice
// of Field objects based on the specified delimiter.
func splitBytes(b []byte, delim byte, fields []Field) error {
	for i := 0; i < len(fields)-1; i++ {
		idx := bytes.IndexByte(b, delim)
		if idx == -1 {
			return fmt.Errorf("Expected []b to contain %v fields using delimiter '%+v'", len(fields), string(delim))
		}
		fields[i].data = b[:idx]
		b = b[idx+1:]
	}
	fields[len(fields)-1].data = b
	return nil
}
