package main

import (
	"fmt"
	"github.com/cet001/hastycsv"
	"strings"
)

func main() {
	r := strings.NewReader(`Honda|Acura NSX|2017|18.1
Chevrolet|Corvette|2016|16.5
BMW|M3|2015|18.7
Audi|A3|2014|25.4`)

	// Create our CSV reader and configure it to use '|' as the field delimiter
	hastyCsvReader := hastycsv.NewReader()
	hastyCsvReader.Comma = '|'

	err := hastyCsvReader.Read(r, func(i int, fields []hastycsv.Field) error {
		fmt.Printf("line %v: make=%v, model=%v, year=%v, mpg=%v\n", i,
			fields[0].String(),
			fields[1].String(),
			fields[2].Uint32(),
			fields[3].Float32(),
		)
		return nil // To halt reading, return an error here.
	})

	if err != nil {
		fmt.Printf("Error parsing csv file: %v\n", err)
	}
}
