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

	filePath := directory + "./Contributors.csv"
	// Creating CSV file
	err := prepareCSV(filePath)

	if err != nil {
		return err
	}

	contributors := merge(authors, reviewers)

	err = writeFile(filePath, contributors)

	if err != nil {
		return err
	}

	return nil
}

// Creates CSV and write header in it.
func prepareCSV(filePath string) error {

	var file, err = os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Headers in CSV file.
	writer.Write([]string{
		"Contributor",
		"Created",
		"Reviewed",
	})

	return err
}

func writeFile(filePath string, contributors []Contributor) error {
	// Open file using Append mode.
	var file, err = os.OpenFile(filePath, os.O_APPEND, 0644)
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

	return err
}

// Appends new name in contributors slice
func appendNew(previous Contributor, name string, contributors []Contributor) (Contributor, []Contributor) {
	if previous.name != name {
		contributors = append(contributors, previous)
		previous = Contributor{}
		previous.name = name
	}
	return previous, contributors
}

// Collects contributors stats from author and reviewers slices
func merge(authors []string, reviewers []string) []Contributor {

	contributors := []Contributor{}
	previous := Contributor{}
	var i, j int

	// Merging and Writing Sorted authors and reviewers in csv file
	for i < len(authors) && j < len(reviewers) {
		if authors[i] < reviewers[j] {
			previous, contributors = appendNew(previous, authors[i], contributors)
			previous.created++
			i++
		} else if authors[i] > reviewers[j] {
			previous, contributors = appendNew(previous, reviewers[j], contributors)
			previous.reviewed++
			j++
		} else {
			previous, contributors = appendNew(previous, authors[i], contributors)
			previous.created++
			previous.reviewed++
			i++
			j++
		}

	}

	if contributors[0].name == "" {
		contributors = contributors[1:]
	}

	// Writing Rest of Contributors in CSV file
	contributorType := "a"
	previous, contributors = copyContributors(contributors, previous, authors[i:], contributorType)

	contributorType = "r"
	previous, contributors = copyContributors(contributors, previous, reviewers[j:], contributorType)

	// Check for last contributor
	if previous.name != contributors[len(contributors)-1].name {
		contributors = append(contributors, previous)
	}

	fmt.Println("Contributors", contributors)
	return contributors
}

// Appends contributors stats from single slice
func copyContributors(contributors []Contributor, previous Contributor, names []string, nameType string) (Contributor, []Contributor) {
	var i = 0
	for ; i < len(names); i++ {
		if i == len(names)-1 && previous.name == names[i] {
			previous.created++
			contributors = append(contributors, previous)
		}

		previous, contributors = appendNew(previous, names[i], contributors)

		if nameType == "a" {
			previous.created++
		} else {
			previous.reviewed++
		}
	}
	return previous, contributors
}
