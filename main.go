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
	ctx, cancel := context.WithTimeout(context.Background(), parameters.timeout)
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
	rootNodeID := getRootNodeID(ctx, domClient)

	branchURL := getBranchURL(ctx, domClient, rootNodeID, parameters.branch)

	// Navigate to branch
	navigate(ctx, c, branchURL)
	rootNodeID = getRootNodeID(ctx, domClient)

	// Get commit code for latest message
	commitCode := parseCommitCode(ctx, domClient, rootNodeID, ".u-monospace.Metadata td", "</td>")
	fmt.Println("Commit Code ", commitCode)

	if parameters.URL[len(parameters.URL)-1] == '/' {
		parameters.URL += "+/"
	} else {
		parameters.URL += "/+/"
	}

	// Creation of directory for commit messages and contributor csv file
	createDir(parameters.commitsDir)

	createDir(parameters.contributorDir)

	// Store contributors
	var authors, reviewers []string

	// Parse commit message, contributors info
	// Navigate to parent commit message
	for i := 1; i <= parameters.commitNum; i++ {
		// Navigate to commit url
		Link := parameters.URL + commitCode
		navigate(ctx, c, Link)
		rootNodeID = getRootNodeID(ctx, domClient)

		// Getting Commit Message and Contributors
		commitMessage, newAuthors, newReviewers := parseMessage(ctx, domClient, rootNodeID)

		// Write Commit Message
		filePath := parameters.commitsDir + "./Commits" + commitCode[0:6] + ".txt"
		WriteMessage(filePath, commitMessage)

		// Store Contributors
		authors = append(authors, newAuthors...)
		reviewers = append(reviewers, newReviewers...)

		// Get next commit code
		search := strings.ReplaceAll(parameters.URL, "https://chromium.googlesource.com", "")

		commitCode = parseCommitCode(ctx, domClient, rootNodeID, "a[href*='"+search+commitCode+"%5E']", "</a>")

		fmt.Println("Commit Code ", commitCode)

	}

	// Merging and Writing contributors in CSV
	Merge.MergeWrite(authors, reviewers, parameters.contributorDir)

	return nil
}

func createDir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModeDir|0755)
	}
}

func isError(err error) {
	if err != nil {
		log.Fatal(err)
	}
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

// Returns document root NodeID
func getRootNodeID(ctx context.Context, domClient cdp.DOM) dom.NodeID {
	doc, err := domClient.GetDocument(ctx, nil)
	isError(err)

	return doc.Root.NodeID
}

// Returns Branch url from list of branch names given on repository page
func getBranchURL(ctx context.Context, domClient cdp.DOM, RootID dom.NodeID, branchName string) string {

	var branchURL string
	QueryNodes := QuerySelectorAll(ctx, domClient, RootID, "ul.RefList-items > li > a")

	// Search in Nodes for branch URL
	for _, nodeId := range QueryNodes.NodeIDs {
		result := GetOuterHTML(ctx, domClient, nodeId)

		if strings.Contains(result.OuterHTML, branchName) {
			branchURL = strings.Split(result.OuterHTML, "\">")[0]
			branchURL = strings.TrimLeft(branchURL, "<a href=\"")
			break
		}
	}

	branchURL = "https://chromium.googlesource.com" + branchURL

	return branchURL
}

// Return commit code as per tag
func parseCommitCode(ctx context.Context, domClient cdp.DOM, NodeID dom.NodeID, selector string, tag string) string {
	html := QueryHTML(ctx, domClient, NodeID, selector)

	commitCode := strings.Split(strings.TrimRight(html, tag), ">")[1]

	return commitCode
}

// Write Commit messages
func WriteMessage(fileName string, commitMessage []string) {
	f, err := os.Create(fileName)

	isError(err)

	defer f.Close()

	for _, message := range commitMessage {
		_, err := f.WriteString(message + "\n")
		isError(err)
	}

}

func QuerySelectorAll(ctx context.Context, domClient cdp.DOM, NodeID dom.NodeID, Selector string) *dom.QuerySelectorAllReply {
	QueryNodes, err := domClient.QuerySelectorAll(ctx, &dom.QuerySelectorAllArgs{
		NodeID:   NodeID,
		Selector: Selector,
	})
	isError(err)
	return QueryNodes
}

func QuerySelector(ctx context.Context, domClient cdp.DOM, NodeID dom.NodeID, Selector string) *dom.QuerySelectorReply {
	QueryNode, err := domClient.QuerySelector(ctx, &dom.QuerySelectorArgs{
		NodeID:   NodeID,
		Selector: Selector,
	})
	isError(err)
	return QueryNode
}

func GetOuterHTML(ctx context.Context, domClient cdp.DOM, NodeId dom.NodeID) *dom.GetOuterHTMLReply {
	result, err := domClient.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
		NodeID: &NodeId,
	})
	isError(err)
	return result
}

// Selects a node using selector and return html output for it
func QueryHTML(ctx context.Context, domClient cdp.DOM, NodeID dom.NodeID, Selector string) string {
	QueryNode := QuerySelector(ctx, domClient, NodeID, Selector)

	result := GetOuterHTML(ctx, domClient, QueryNode.NodeID)

	return result.OuterHTML
}

// Pareses message and return commit message and contributors stat
func parseMessage(ctx context.Context, domClient cdp.DOM, NodeID dom.NodeID) ([]string, []string, []string) {

	// Get complete commit message
	rawMessage := QueryHTML(ctx, domClient, NodeID, ".MetadataMessage")

	commitMessage := getCommitMessage(rawMessage)

	authors, reviewers := parseContributorStat(commitMessage)

	return commitMessage, authors, reviewers
}

// Modifies raw message and returns commit message from it
func getCommitMessage(rawMessage string) []string {
	// Convert HTML characters
	r := strings.NewReplacer("&lt;", "<", "&gt;", ">")

	commitMessage := strings.Split(r.Replace(rawMessage), "\n")

	commitMessage[0] = strings.Split(commitMessage[0], ">")[1]
	commitMessage = commitMessage[:len(commitMessage)-1]

	return commitMessage
}

// Loops thorugh commit message and return contributors stats
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
