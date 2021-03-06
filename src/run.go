package src

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/rpcc"
)

// Structure for CLI arguments
type Parameters struct {
	URL, Branch, CommitsDir, ContributorDir string
	CommitNum                               int
	Timeout                                 time.Duration
}

func Run(parameters Parameters) error {
	ctx, cancel := context.WithTimeout(context.Background(), parameters.Timeout)
	defer cancel()

	// Devtools package used for finding websocket URL
	devt := devtool.New("http://127.0.0.1:9222")
	pt, err := devt.Get(ctx, devtool.Page)
	if err != nil {
		pt, err = devt.Create(ctx)
		isError(err)
	}

	// Initiate a new RPC connection to the Chrome DevTools Protocol target.
	conn, err := rpcc.DialContext(ctx, pt.WebSocketDebuggerURL)
	isError(err)

	defer conn.Close() // Leaving connections open will leak memory.

	c := cdp.NewClient(conn)
	domClient := c.DOM

	// Navigate to repository
	navigate(ctx, c, parameters.URL)
	rootNodeID := GetRootNodeID(ctx, domClient)

	branchURL := getBranchURL(ctx, domClient, rootNodeID, parameters.Branch)

	// Navigate to branch
	navigate(ctx, c, branchURL)
	rootNodeID = GetRootNodeID(ctx, domClient)

	// Get commit code for latest message
	commitCode := InnerHTML(ctx, domClient, rootNodeID, ".u-monospace.Metadata td", "</td>")
	fmt.Println("Commit Code ", commitCode)

	if parameters.URL[len(parameters.URL)-1] == '/' {
		parameters.URL += "+/"
	} else {
		parameters.URL += "/+/"
	}

	searchURL := strings.ReplaceAll(parameters.URL, "https://chromium.googlesource.com", "")

	// Creation of directory for commit messages and contributor csv file
	createDir(parameters.CommitsDir)
	createDir(parameters.ContributorDir)

	// Store contributors
	var authors, reviewers []string

	// Parse commit message, contributors info
	// Navigate to parent commit message
	for i := 1; i <= parameters.CommitNum; i++ {
		// Navigate to commit url
		Link := parameters.URL + commitCode
		navigate(ctx, c, Link)
		rootNodeID = GetRootNodeID(ctx, domClient)

		// Getting Commit Message and Contributors
		commitMessage, newAuthors, newReviewers := parseMessage(ctx, domClient, rootNodeID)

		// Write Commit Message
		WriteMessage(parameters.CommitsDir, commitCode[0:6], commitMessage)

		// Store Contributors
		authors = append(authors, newAuthors...)
		reviewers = append(reviewers, newReviewers...)

		// Get next commit code
		commitCode = InnerHTML(ctx, domClient, rootNodeID, "a[href*='"+searchURL+commitCode+"%5E']", "</a>")
		fmt.Println("Commit Code ", commitCode)

	}

	// Writing contributors in CSV
	WriteContributors(authors, reviewers, parameters.ContributorDir)

	return nil
}

// navigate to the URL and wait for DOMContentEventFired
func navigate(ctx context.Context, client *cdp.Client, URL string) {

	// Enable events on the Page domain
	err := client.Page.Enable(ctx)
	isError(err)

	// Open a DOMContentEventFired client to buffer this event.
	domContent, err := client.Page.DOMContentEventFired(ctx)
	isError(err)
	defer domContent.Close()

	nav, err := client.Page.Navigate(ctx, page.NewNavigateArgs(URL))
	isError(err)

	_, err = domContent.Recv()
	isError(err)

	fmt.Printf("Page loaded with frame ID: %s\n", nav.FrameID)
}

// Write Commit messages
func WriteMessage(CommitsDir string, CommitCode string, commitMessage []string) {
	filePath := CommitsDir + "Commits" + CommitCode + ".txt"

	err := CreateFile(filePath)
	isError(err)

	f, err := os.OpenFile(filePath, os.O_WRONLY, 0644)
	isError(err)

	defer f.Close()

	for _, message := range commitMessage {
		_, err := f.WriteString(message + "\n")
		isError(err)
	}

}

// Creates File.
func CreateFile(filePath string) error {

	var file, err = os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	file.Name()
	return nil
}

func isError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func createDir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModeDir|0755)
	}
}
