package src

import (
	"encoding/csv"
	"os"
	"strconv"
)

type Contributor struct {
	created  int
	reviewed int
}

// Main function to initalize varibale for use for merging and writing Contributors
func WriteContributors(authors []string, reviewers []string, directory string) error {

	filePath := directory + "./Contributors.csv"
	// Creating CSV file
	err := CreateFile(filePath)

	if err != nil {
		return err
	}

	contributors := mergeContributors(authors, reviewers)

	err = writeMap(filePath, contributors)

	if err != nil {
		return err
	}

	return nil
}

// Collects contributors stats from author and reviewers slices
func mergeContributors(authors []string, reviewers []string) map[string]Contributor {

	contributors := make(map[string]Contributor)

	for i := 0; i < len(authors); i++ {
		contributors[authors[i]] = Contributor{created: contributors[authors[i]].created + 1, reviewed: contributors[authors[i]].reviewed}
	}
	for i := 0; i < len(reviewers); i++ {
		contributors[reviewers[i]] = Contributor{created: contributors[reviewers[i]].created, reviewed: contributors[reviewers[i]].reviewed + 1}
	}

	return contributors
}

func writeMap(filePath string, contributors map[string]Contributor) error {
	// Open file using Write mode.
	var file, err = os.OpenFile(filePath, os.O_WRONLY, 0644)
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

	// Writing contributors stat line by line
	for key, value := range contributors {
		writer.Write([]string{
			key,
			strconv.Itoa(value.created),
			strconv.Itoa(value.reviewed),
		})
	}

	return err
}
