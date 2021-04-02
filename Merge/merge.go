package Merge

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

type Contributor struct {
	name     string
	created  int
	reviewed int
}

// Main function to initalize varibale for use for merging and writing Contributors
func MergeWrite(authors []string, reviewers []string, directory string) error {
	previous := Contributor{}

	if authors[0] > reviewers[0] {
		previous.name = reviewers[0]
	} else {
		previous.name = authors[0]
	}

	// Creating CSV file
	file, err := os.Create(directory + "./Contributors.csv")

	if err != nil {
		return err
	}

	defer file.Close()

	writer := csv.NewWriter(file)

	defer writer.Flush()

	// Headers in CSV file
	writer.Write([]string{
		"contributor",
		"created",
		"reviewed",
	})
	merge(authors, reviewers, previous, writer)
	return nil
}

// Writes contributor in CSV files
func write(previous Contributor, writer *csv.Writer) {
	if previous.name == "" {
		return
	}
	fmt.Println("Writing", previous.name, previous.created, previous.reviewed)
	writer.Write([]string{
		previous.name,
		strconv.Itoa(previous.created),
		strconv.Itoa(previous.reviewed),
	})
}

// Helper function to compare Contributors
// Write in CSV file if Contributors name not equal
func compare(previous Contributor, contributor string, writer *csv.Writer) Contributor {
	if previous.name != contributor {
		// Write the contributor and create new contributor
		write(previous, writer)
		previous = Contributor{}
		previous.name = contributor
	}
	return previous
}

func merge(authors []string, reviewers []string, previous Contributor, writer *csv.Writer) {

	var i, j int
	// Merging and Writing Sorted authors and reviewers in csv file
	for i < len(authors) && j < len(reviewers) {
		if authors[i] < reviewers[j] {
			previous = compare(previous, authors[i], writer)
			previous.created++
			i++
		} else if authors[i] > reviewers[j] {
			previous = compare(previous, reviewers[j], writer)
			previous.reviewed++
			j++
		} else {
			previous = compare(previous, authors[i], writer)
			previous.created++
			previous.reviewed++
			i++
			j++
		}

	}
	write(previous, writer)
	previous = Contributor{}

	// Writing Rest of Contributors in CSV file
	for ; i < len(authors); i++ {
		if i == len(authors)-1 && previous.name == authors[i] {
			previous.created++
			write(previous, writer)
		}

		previous = compare(previous, authors[i], writer)
		previous.created++
	}

	for ; j < len(reviewers); j++ {
		if j == len(reviewers)-1 && previous.name == reviewers[j] {
			previous.reviewed++
			write(previous, writer)
		}

		previous = compare(previous, reviewers[j], writer)
		previous.reviewed++
	}
}
