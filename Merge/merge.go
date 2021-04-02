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
	// Creating CSV file
	err := prepareCSV(directory + "./Contributors.csv")

	if err != nil {
		return err
	}

	contributors := merge(authors, reviewers)

	err = writeFile(directory+"./Contributors.csv", contributors)

	if err != nil {
		return err
	}

	return nil
}

func writeFile(path string, contributors []Contributor) error {
	// Open file using READ & WRITE permission.
	var file, err = os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write some text line-by-line to file.
	for _, contributor := range contributors {
		writer.Write([]string{
			contributor.name,
			strconv.Itoa(contributor.created),
			strconv.Itoa(contributor.reviewed),
		})

	}

	// Save file changes.
	err = file.Sync()
	if err != nil {
		return err
	}

	return err
}

func prepareCSV(filePath string) error {

	var file, err = os.Create(filePath)
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
	file.Sync()

	return err
}

// Helper function to compare Contributors
// Write in CSV file if Contributors name not equal

func compare(previous Contributor, contributor string) Contributor {
	if previous.name != contributor {
		// Write the contributor and create new contributor
		previous = Contributor{}
		previous.name = contributor
	}
	return previous
}

func merge(authors []string, reviewers []string) []Contributor {

	contributors := []Contributor{}
	previous := Contributor{}

	if authors[0] > reviewers[0] {
		previous.name = reviewers[0]
	} else {
		previous.name = authors[0]
	}

	var i, j int
	// Merging and Writing Sorted authors and reviewers in csv file
	for i < len(authors) && j < len(reviewers) {
		if authors[i] < reviewers[j] {
			current := compare(previous, authors[i])
			if previous != current {
				contributors = append(contributors, previous)
				previous = current
			}
			previous.created++
			i++
		} else if authors[i] > reviewers[j] {

			current := compare(previous, reviewers[j])
			if previous != current {
				contributors = append(contributors, previous)
				previous = current
			}
			previous.reviewed++
			j++
		} else {
			current := compare(previous, authors[i])
			if previous != current {
				contributors = append(contributors, previous)
				previous = current
			}
			previous.created++
			previous.reviewed++
			i++
			j++
		}

	}

	// Writing Rest of Contributors in CSV file
	for ; i < len(authors); i++ {
		if i == len(authors)-1 && previous.name == authors[i] {
			previous.created++
			contributors = append(contributors, previous)
		}

		current := compare(previous, authors[i])
		if previous != current {
			contributors = append(contributors, previous)
			previous = current
		}
		previous.created++
	}

	for ; j < len(reviewers); j++ {
		if j == len(reviewers)-1 && previous.name == reviewers[j] {
			previous.reviewed++
			contributors = append(contributors, previous)
		}

		current := compare(previous, reviewers[j])
		if previous != current {
			contributors = append(contributors, previous)
			previous = current
		}
		previous.reviewed++
	}
	fmt.Println("Contributors ", contributors)
	return contributors
}
