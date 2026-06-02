package mytools
// package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"log"
	// "strings"
)

// User represents a single record in the CSV file.
// The fields are named after the CSV header columns for clarity.
type DeliveryInfo struct {
	FirstName string
	LastName  string
	Number string
	Status string
	StreetAddress string
	City string
}

const DELIVERY_FILE = "deliveries.csv"
var deliveryData []DeliveryInfo

func ReadCSV() {
	// 1. Sample CSV data. In a real application, you would use os.Open("yourfile.csv").
	file, err := os.Open(DELIVERY_FILE)
	if err != nil {panic(err)}
	defer file.Close()

	// 2. Read the header row. This is important to skip it in the loop.
	reader := csv.NewReader(file)

	// r0 = make([]byte, 200)
	header, err := reader.Read()
	if err != nil {
		log.Fatalf("Failed to read header row: %v", err)
	}
	fmt.Printf("File header: %v\n", header)

	// This slice will hold all of our User records.
	var infos []DeliveryInfo

	// 3. Loop through the remaining records.
	for {
		// Read one record (a slice of strings).
		record, err := reader.Read()
		if err == io.EOF {
			// End of file is reached, break the loop.
			break
		}
		if err != nil {
			log.Fatalf("Error reading record: %v", err)
		}

		info := DeliveryInfo{
			FirstName:    record[0],
			LastName: record[1],
			Number: record[2],
			Status: record[3],
			StreetAddress: record[4],
			City: record[5],
		}

		// Add the new User struct to our slice.
		infos = append(infos, info)
	}
	deliveryData = infos

	// 5. Print the final slice of structs to show the result.
	// The %+v format verb prints both the field names and their values.
	fmt.Println("\n--- Successfully parsed data into Go structs ---")
	for _, info := range infos {
		fmt.Printf("%s - %s - %s - %s\n", info.FirstName, info.LastName, info.Number, info.City)
	}
}


// return package delivery information, from package number
func FindPackageByNumber(nb string) string {
	// read data from file, if not already done
	fmt.Printf("Calling  FindPackageByNumber(%s)\n", nb)
	if deliveryData == nil {ReadCSV()}
	if deliveryData != nil {
		for _, info := range deliveryData {
			if nb == info.Number {
				res := fmt.Sprintf("Delivery information: LastName=%s, Number=%s, Status=%s, Address=%s, City=%s", info.LastName, info.Number, info.Status, info.StreetAddress, info.City)
				fmt.Println("returning: ", res)
				return res
			}
		}
	}
	notFound := "Package cannot be found"
	return notFound
}

/*
func main() {
	ReadCSV()
}
*/
