/*
	Package main implements scrapping of Git repositories on chromium for
	storing commit messages and information about contributors
*/
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Vishalghyv/Commit-Messages/Merge"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/dom"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/rpcc"
)

// Structure for CLI arguments
type Parameters struct {
	URL, branch, commitsDir, contributorDir string
	commitNum                               int
	timeout                                 time.Duration
}

var (
	ctx context.Context
	c   *cdp.Client
)

func main() {
	var para = Parameters{}

	// Command Line Arguments
	flag.IntVar(&para.commitNum, "commit-num", 10, "number of last commit messages")
	flag.StringVar(&para.URL, "url", "https://chromium.googlesource.com/chromiumos/platform/tast-tests/", "Repository URL")
	flag.StringVar(&para.branch, "branch", "main", "Branch Name")
	flag.DurationVar(&para.timeout, "timeout", 10*time.Second, "Maximum time program to run")
	flag.StringVar(&para.commitsDir, "commits-dir", "./Commits/", "Commit message folder path")
	flag.StringVar(&para.contributorDir, "contributor-dir", "./Commits", "Contributor CSV folder path")
	flag.Parse()

	err := run(para)
	if err != nil {
		fmt.Println(err)
	}
}

func run(parameters Parameters) error {
	// Context
	ctx, cancel := context.WithTimeout(context.Background(), parameters.timeout)
	defer cancel()

	// Port used
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

	// Global cdp.Client is defined - Important to not redefine it
	c = cdp.NewClient(conn)

	// Navigate to repository
	Link := parameters.URL
	doc := OpenDoc(c, Link, "https://google.com")

	branchURL := firstCommmitURL(doc.Root.NodeID, parameters.branch)

	// Navigate to branch
	Link = "https://chromium.googlesource.com" + branchURL

	doc = OpenDoc(c, Link, "https://google.com")

	// Get commit code for latest message
	commitCode := parseCommitCode(doc.Root.NodeID, ".u-monospace.Metadata td", "</td>")

	fmt.Println("Commit Code ", commitCode)

	if parameters.URL[len(parameters.URL)-1] == '/' {
		parameters.URL += "+/"
	} else {
		parameters.URL += "/+/"
	}

	// Creation of directory for commit messages and contributor csv file

	if _, err := os.Stat(parameters.commitsDir); os.IsNotExist(err) {
		os.Mkdir(parameters.commitsDir, os.ModeDir|0755)
	}

	if _, err := os.Stat(parameters.contributorDir); os.IsNotExist(err) {
		os.Mkdir(parameters.contributorDir, os.ModeDir|0755)
	}

	// Store parsed contributors
	var authors, reviewers []string

	// Parse commit message, contributors info
	// Navigate to parent commit message
	for i := 1; i <= parameters.commitNum; i++ {
		// Navigate to commit code url
		Link := parameters.URL + commitCode

		doc = OpenDoc(c, Link, "https://google.com")

		// Getting Commit Message and Contributors
		commitMessage, newAuthors, newReviewers := parseMessage(doc.Root.NodeID)

		// Storing Contributors
		authors = append(authors, newAuthors...)
		reviewers = append(reviewers, newReviewers...)

		// Writing Commit Message
		textFile := parameters.commitsDir + "./Commits" + commitCode[0:6] + ".txt"
		WriteMessage(textFile, commitMessage)

		// Getting next commit code
		search := strings.ReplaceAll(parameters.URL, "https://chromium.googlesource.com", "")

		commitCode = parseCommitCode(doc.Root.NodeID, "a[href*='"+search+commitCode+"%5E']", "</a>")

		fmt.Println("Commit Code ", commitCode)

	}

	// Sort Authors and Reviewers
	sort.Strings(authors)
	sort.Strings(reviewers)

	// Merging and Writing contributors in CSV
	Merge.MergeWrite(authors, reviewers, parameters.contributorDir)

	return nil
}

func isError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Function to navigate to a URL and return opened Document
func OpenDoc(client *cdp.Client, URL string, Referrer string) *dom.GetDocumentReply {
	// Create the Navigate arguments with the optional Referrer field set.
	navArgs := page.NewNavigateArgs(URL).
		SetReferrer(Referrer)

	nav, err := client.Page.Navigate(ctx, navArgs)
	isError(err)

	// Open a DOMContentEventFired client to buffer this event.
	domContent, err := client.Page.DOMContentEventFired(ctx)
	isError(err)
	defer domContent.Close()

	// Enable events on the Page domain
	err = client.Page.Enable(ctx)
	isError(err)

	// Wait until we have a DOMContentEventFired event.
	_, err = domContent.Recv()
	isError(err)

	fmt.Printf("Page loaded with frame ID: %s\n", nav.FrameID)

	doc, err := client.DOM.GetDocument(ctx, nil)
	isError(err)

	return doc
}

func firstCommmitURL(NodeID dom.NodeID, branchName string) string {
	// Parse url for branch

	// Get the outer HTML for the page.
	QueryNodes := QuerySelectorAll(NodeID, "ul.RefList-items > li > a")

	var branchURL string

	// Search in Node for branch URL
	for _, nodeId := range QueryNodes.NodeIDs {
		result := GetOuterHTML(nodeId)

		if strings.Contains(result.OuterHTML, branchName) {
			branchURL = strings.Split(result.OuterHTML, "\">")[0]
			branchURL = strings.TrimLeft(branchURL, "<a href=\"")
			break
		}
	}

	return branchURL
}

func parseCommitCode(NodeID dom.NodeID, selector string, tag string) string {
	html := QueryHTML(NodeID, selector)

	commitCode := strings.Split(strings.TrimRight(html, tag), ">")[1]

	return commitCode
}

func WriteMessage(fileName string, commitMessage []string) {
	f, err := os.Create(fileName)

	isError(err)

	defer f.Close()
	for _, message := range commitMessage {
		f.WriteString(message)
		f.WriteString("\n")
	}

}

func QuerySelectorAll(NodeID dom.NodeID, Selector string) *dom.QuerySelectorAllReply {
	QueryNodes, err := c.DOM.QuerySelectorAll(ctx, &dom.QuerySelectorAllArgs{
		NodeID:   NodeID,
		Selector: Selector,
	})
	isError(err)
	return QueryNodes
}

func QuerySelector(NodeID dom.NodeID, Selector string) *dom.QuerySelectorReply {
	QueryNode, err := c.DOM.QuerySelector(ctx, &dom.QuerySelectorArgs{
		NodeID:   NodeID,
		Selector: Selector,
	})
	isError(err)
	return QueryNode
}

func GetOuterHTML(NodeId dom.NodeID) *dom.GetOuterHTMLReply {
	result, err := c.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
		NodeID: &NodeId,
	})
	isError(err)
	return result
}

// Function to select a node and return html output for it
func QueryHTML(NodeID dom.NodeID, Selector string) string {
	QueryNode := QuerySelector(NodeID, Selector)

	result := GetOuterHTML(QueryNode.NodeID)

	return result.OuterHTML
}

func parseMessage(NodeID dom.NodeID) ([]string, []string, []string) {

	// Get complete commit message
	rawMessage := QueryHTML(NodeID, ".MetadataMessage")

	// Convert HTML characters
	r := strings.NewReplacer("&lt;", "<", "&gt;", ">")

	commitMessage := strings.Split(r.Replace(rawMessage), "\n")

	commitMessage[0] = strings.Split(commitMessage[0], ">")[1]
	commitMessage = commitMessage[:len(commitMessage)-1]

	authors, reviewers := parseContributorStat(commitMessage)

	return commitMessage, authors, reviewers
}

func parseContributorStat(commitMessage []string) ([]string, []string) {
	var authors, reviewers []string

	for _, value := range commitMessage {
		if strings.Contains(value, "Tested-by:") && (value[0] == ' ' || value[0] == 'T') {
			authors = append(authors, strings.Split(value, "Tested-by: ")[1])
		}

		if strings.Contains(value, "Reviewed-by:") && (value[0] == ' ' || value[0] == 'R') {
			reviewers = append(reviewers, strings.Split(value, "Reviewed-by: ")[1])
		}
	}

	return authors, reviewers
}
