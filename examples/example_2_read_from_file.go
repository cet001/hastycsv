package main

import (
	"fmt"
	"github.com/cet001/hastycsv"
)

func main() {
	const csvFile = "./examples/sample_data.csv"

	err := hastycsv.ReadFile(csvFile, '|', func(lineNum int, fields []hastycsv.Field) {
		if lineNum == 1 {
			return
		} // skip header record

		fmt.Printf("line %v: make=%v, model=%v, year=%v, mpg=%v\n", lineNum,
			fields[0].String(),
			fields[1].String(),
			fields[2].Uint32(),
			fields[3].Float32(),
		)
	})

	if err != nil {
		fmt.Printf("Error parsing csv file: %v\n", err)
	}
}
