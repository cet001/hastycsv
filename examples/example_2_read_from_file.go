package main

import (
	"fmt"
	"github.com/cet001/hastycsv"
)

func main() {
	const csvFile = "./examples/sample_data.csv"

	err := hastycsv.ReadFile(csvFile, '|', func(i int, fields []hastycsv.Field) {
		fmt.Printf("line %v: make=%v, model=%v, year=%v, mpg=%v\n", i,
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
