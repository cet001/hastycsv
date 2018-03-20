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
	"os"
	"strconv"
	"unsafe"
)

// Needed by Field.Uint32() parser
var base10exp = []uint32{1, 10, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000, 1000000000}

// Reads records from a CSV-encoded file or io.Reader.
type Reader struct {
	// Delimiter is the CSV field delimiter. It is set to ',' by NewReader().
	Delimiter byte
	fields    []Field
	Row       int
	err       error
}

// Returns a new Reader whose Delimiter is set to the comma character (',').
func NewReader() *Reader {
	return &Reader{
		Delimiter: ',',
	}
}

func (me *Reader) Read(r io.Reader, nextRecord func(i int, record []Field)) error {
	var fields []Field
	isFirstRecord := true
	delim := me.Delimiter
	me.Row = 0

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
			continue
		}

		me.Row++

		if err := splitBytes(b, delim, fields); err != nil {
			return fmt.Errorf("Line %v: %v: \"%v\"", me.Row, err, string(b))
		}

		nextRecord(me.Row, fields)

		if me.err != nil {
			return fmt.Errorf("Line %v: %v", me.Row, me.err)
		}
	}

	if me.err != nil {
		return fmt.Errorf("Line %v: %v", me.Row, me.err)
	}

	if err := lineScanner.Err(); err != nil {
		return fmt.Errorf("Error scanning input: %v", err)
	}

	return nil
}

func ReadFile(csvFilePath string, delim byte, nextRecord func(i int, record []Field)) error {
	f, err := os.Open(csvFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	r := NewReader()
	r.Delimiter = delim
	return r.Read(f, nextRecord)
}

// Represents a field (encoded as a UTF-8 string) within a CSV record.
type Field struct {
	reader *Reader
	data   []byte
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
	v := uint32(0)
	d := len(me.data)
	for _, b := range me.data {
		if b < '0' || b > '9' {
			if me.reader.err == nil {
				me.reader.err = fmt.Errorf("Field \"%v\" contains non-numeric character '%v'", string(me.data), string(b))
			}
			return 0
		}
		d--
		v += uint32(b-'0') * base10exp[d]
	}
	return v
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
