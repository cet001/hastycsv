package hastycsv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Test helper
func makeField(s string) Field {
	return Field{reader: NewReader(), data: []byte(s)}
}

func TestField_ToLower(t *testing.T) {
	values := []string{
		"",
		"abc",
		"ABC",
		"AbC",
		"!ABC-123?",
		"!@#$%^&*()_+",
	}

	for i, value := range values {
		assert.Equal(t,
			strings.ToLower(value),
			makeField(value).ToLower().String(),
			"values[%v]", i,
		)
	}
}

func TestField_String(t *testing.T) {
	values := []string{
		"",
		" ",
		"a",
		"abcdefg",
		"ABC123",
	}

	for _, s := range values {
		field := makeField(s)
		assert.Equal(t, s, field.String())
	}
}

func TestField_Uint32(t *testing.T) {
	testValues := map[string]uint32{
		"0":          0,
		"000":        0,
		"1":          1,
		"12":         12,
		"12345678":   12345678,
		"012":        12,
		"4294967295": 4294967295, // max 32-bit unsigned int
	}

	for testValue, expectedValue := range testValues {
		field := makeField(testValue)
		actualValue := field.Uint32()
		assert.Nil(t, field.reader.err)
		assert.Equal(t, expectedValue, actualValue)
	}
}

func TestField_Uint32_parseError(t *testing.T) {
	badlyFormattedInts := []string{
		"-1",
		"-1.23",
		"1.5",
		"1F",
		"x",
		"abc",
		" ",
	}

	for _, badlyFormattedInt := range badlyFormattedInts {
		field := makeField(badlyFormattedInt)
		assert.Equal(t, uint32(0), field.Uint32())
		assert.NotNil(t, field.reader.err, `value="%v"`, badlyFormattedInt)
	}
}

func TestField_Float32(t *testing.T) {
	testValues := map[string]float32{
		"0":     0,
		"0.0":   0,
		"1":     1,
		"0.125": 0.125,
		".125":  0.125,
		"1.25":  1.25,
	}

	for testValue, expectedValue := range testValues {
		field := makeField(testValue)
		actualValue := field.Float32()
		assert.Nil(t, field.reader.err)
		assert.Equal(t, expectedValue, actualValue)
	}
}

func TestField_Float32_parseError(t *testing.T) {
	badlyFormattedFloats := []string{
		"x",
		"",
		" ",
		"1.2.3",
	}

	for _, badlyFormattedFloat := range badlyFormattedFloats {
		field := makeField(badlyFormattedFloat)
		assert.Equal(t, float32(0), field.Float32())
		assert.NotNil(t, field.reader.err)
	}
}

func TestSplitBytes(t *testing.T) {
	testData := []string{
		"",
		"foo",
		"foo,bar",
		"foo,bar,baz",
		"a,b,c",
		",two,three",
		"one,two,",
		",,",
		"aa,bb,cc,dd,ee,ff,gg,",
	}

	toStringSlice := func(fields []Field) []string {
		s := make([]string, 0, len(fields))
		for _, field := range fields {
			s = append(s, field.String())
		}
		return s
	}

	for _, s := range testData {
		expectedSplit := strings.Split(s, ",")
		record := make([]Field, len(expectedSplit))
		splitBytes([]byte(s), ',', record)
		assert.Equal(t, expectedSplit, toStringSlice(record))
	}
}

// Special case: split bytes into a record that contains only 1 field.  In this
// case, even if the input string contains the delimiter field, the entire string
// should get assigned to the record's single field.
func TestSplitBytes_recordWithOnlyOneField(t *testing.T) {
	record := make([]Field, 1)
	splitBytes([]byte("foo|bar"), '|', record)
	assert.Equal(t, "foo|bar", record[0].String())
}

// Create a 3-field record and attempt to split a line with no field delimiters
func TestSplitBytes_wrongFieldCount(t *testing.T) {
	record := make([]Field, 3)
	assert.NotNil(t, splitBytes([]byte("blah"), '|', record))
}

func TestRead(t *testing.T) {
	type Person struct {
		name   string
		age    uint32
		weight float32
	}

	persons := []Person{
		{name: "bill", age: 30, weight: 154.5},
		{name: "mary", age: 35, weight: 125.1},
	}

	personRecords := []string{}
	for _, p := range persons {
		personRecords = append(personRecords, fmt.Sprintf("%v|%v|%v", p.name, p.age, p.weight))
	}
	in := strings.NewReader(strings.Join(personRecords, "\n"))

	r := NewReader()
	r.Comma = '|'
	err := r.Read(in, func(i int, fields []Field) {
		expectedPerson := persons[i-1]
		assert.Equal(t, expectedPerson.name, fields[0].String())
		assert.Equal(t, expectedPerson.age, fields[1].Uint32())
		assert.Equal(t, expectedPerson.weight, fields[2].Float32())
	})

	assert.Nil(t, err)
}

func TestRead_InvalidComma(t *testing.T) {
	r := NewReader()
	in := strings.NewReader(`10|20|30`)

	for _, invalidCommaChar := range []byte{'\r', '\n'} {
		r.Comma = invalidCommaChar
		err := r.Read(in, func(i int, record []Field) { /* no-op */ })
		assert.EqualError(t, err, `Comma delimiter cannot be \r or \n`)
	}
}

func TestRead_parsingError(t *testing.T) {
	// Create CSV input stream in which line 1 contains an unparseable Uint32 field.
	in := strings.NewReader(`John|123xyz|12.5
Mary|25|130.5`)

	r := NewReader()
	r.Comma = '|'
	err := r.Read(in, func(i int, fields []Field) {
		fields[0].String()
		fields[1].Uint32() // This call will halt csv reading and return an error in the 1st line
		fields[2].Float32()
	})

	assert.EqualError(t, err, "Line 1: Field \"123xyz\" contains non-numeric character 'x'")
}

func TestReadFile(t *testing.T) {
	// Create a temp csv file and add a header plus 2 records.
	tmpCsvFile, err := ioutil.TempFile("", "TestReadRecords")
	if err != nil {
		assert.Fail(t, "Error creating temp file: %v", err)
	}
	defer os.Remove(tmpCsvFile.Name()) // delete the temp file when this function exits

	fmt.Fprintln(tmpCsvFile, "mary,jones,35")    // row 1
	fmt.Fprintln(tmpCsvFile, "bill,anderson,40") // row 2

	err = ReadFile(tmpCsvFile.Name(), ',', func(i int, rec []Field) {
		assert.Equal(t, 3, len(rec))
		switch i {
		case 1:
			assert.Equal(t, "mary", rec[0].String())
			assert.Equal(t, "jones", rec[1].String())
			assert.Equal(t, "35", rec[2].String())
		case 2:
			assert.Equal(t, "bill", rec[0].String())
			assert.Equal(t, "anderson", rec[1].String())
			assert.Equal(t, "40", rec[2].String())
		default:
			assert.Fail(t, "unexpected row index: %v", i)
		}
	})

	assert.Nil(t, err)
}

func TestReadFile_nonexistentFile(t *testing.T) {
	err := ReadFile("NONEXISTENT_FILE.TXT", ',', func(i int, rec []Field) {})
	assert.NotNil(t, err)
}

var tmpString string
var tmpUint32 uint32

func BenchmarkRead_stringValues(b *testing.B) {
	buf := createCsvRecords()
	r := strings.NewReader(buf.String())

	csvReader := NewReader()
	csvReader.Comma = '|'

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		r.Reset(buf.String())
		count := 0
		err := csvReader.Read(r, func(i int, fields []Field) {
			for _, field := range fields {
				tmpString = field.String()
			}
			count++
		})
		assert.Nil(b, err)
	}
}

func BenchmarkRead_intValues(b *testing.B) {
	buf := createCsvRecords()
	r := strings.NewReader(buf.String())

	csvReader := NewReader()
	csvReader.Comma = '|'

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		r.Reset(buf.String())
		count := 0
		err := csvReader.Read(r, func(i int, fields []Field) {
			for _, field := range fields {
				tmpUint32 = field.Uint32()
			}
			count++
		})
		assert.Nil(b, err)
	}
}

func BenchmarkGoCsv_Read_stringValues(b *testing.B) {
	buf := createCsvRecords()
	r := strings.NewReader(buf.String())

	golangReader := csv.NewReader(r)
	golangReader.Comma = '|'
	golangReader.ReuseRecord = true

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		r.Reset(buf.String())
		count := 0
		for {
			fields, err := golangReader.Read()
			if err == io.EOF {
				break
			}
			require.Nil(b, err)
			for _, field := range fields {
				tmpString = field
			}
			count++
		}
	}
}

func BenchmarkGoCsv_Read_intValues(b *testing.B) {
	buf := createCsvRecords()
	r := strings.NewReader(buf.String())

	golangReader := csv.NewReader(r)
	golangReader.Comma = '|'
	golangReader.ReuseRecord = true

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		r.Reset(buf.String())
		count := 0
		for {
			fields, err := golangReader.Read()
			if err == io.EOF {
				break
			}
			require.Nil(b, err)
			for _, field := range fields {
				v, err := strconv.Atoi(field)
				require.Nil(b, err)
				tmpUint32 = uint32(v)
			}
			count++
		}
	}
}

func createCsvRecords() *bytes.Buffer {
	const recordCount = 1000000
	const fieldCount = 5
	const baseValue = 1000000
	record := make([]string, fieldCount)

	buf := bytes.NewBuffer(make([]byte, 0, recordCount))

	for i := 0; i < recordCount; i++ {
		if i > 0 {
			buf.WriteString("\n")
		}

		for j := 0; j < fieldCount; j++ {
			record[j] = fmt.Sprintf("%v", baseValue+i)
		}
		buf.WriteString(strings.Join(record, "|"))
	}

	return buf
}
