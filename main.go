/*
	Package main implements scrapping of Git repositories on chromium for
	storing commit messages and information about contributors
*/
package main

import (
	"context"
	"flag"
	"fmt"
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
		if err != nil {
			return err
		}
	}

	// Initiate a new RPC connection to the Chrome DevTools Protocol target.
	conn, err := rpcc.DialContext(ctx, pt.WebSocketDebuggerURL)
	if err != nil {
		return err
	}
	defer conn.Close() // Leaving connections open will leak memory.

	// Global cdp.Client is defined - Important to not redefine it
	c = cdp.NewClient(conn)

	// Open a DOMContentEventFired client to buffer this event.
	domContent, err := c.Page.DOMContentEventFired(ctx)
	if err != nil {
		return err
	}
	defer domContent.Close()

	// Enable events on the Page domain, it's often preferrable to create
	// event clients before enabling events so that we don't miss any.
	if err = c.Page.Enable(ctx); err != nil {
		return err
	}

	// Function to navigate to a URL and return opened Document
	OpenDoc := func(URL string, Referrer string) (*dom.GetDocumentReply, error) {
		// Create the Navigate arguments with the optional Referrer field set.
		navArgs := page.NewNavigateArgs(URL).
			SetReferrer(Referrer)

		nav, err := c.Page.Navigate(ctx, navArgs)
		if err != nil {
			return nil, err
		}

		// Wait until we have a DOMContentEventFired event.
		if _, err = domContent.Recv(); err != nil {
			return nil, err
		}
		fmt.Printf("Page loaded with frame ID: %s\n", nav.FrameID)

		doc, err := c.DOM.GetDocument(ctx, nil)

		return doc, err
	}

	// Navigate to repository
	Link := parameters.URL
	doc, err := OpenDoc(Link, "https://google.com")
	if err != nil {
		return err
	}

	// Parse url for branch

	// Get the outer HTML for the page.
	QueryNodes, err := QuerySelectorAll(doc.Root.NodeID, "ul.RefList-items > li > a")

	if err != nil {
		return err
	}
	var branch string

	// Search in Node for branch URL
	for _, nodeId := range QueryNodes.NodeIDs {
		result, err := GetOuterHTML(nodeId)

		if err != nil {
			return err
		}

		if strings.Contains(result.OuterHTML, parameters.branch) {
			branch = strings.Split(result.OuterHTML, "\">")[0]
			branch = strings.TrimLeft(branch, "<a href=\"")
			break
		}
	}

	// Navigate to branch
	Link = "https://chromium.googlesource.com" + branch

	doc, err = OpenDoc(Link, "https://google.com")
	if err != nil {
		return err
	}

	// Get commit code for latest message
	html, err := QueryHTML(doc.Root.NodeID, ".u-monospace.Metadata td")

	if err != nil {
		return err
	}

	commitCode := strings.Split(strings.TrimRight(html, "</td>"), ">")[1]
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

		doc, err = OpenDoc(Link, "https://google.com")
		if err != nil {
			return err
		}
		// Get complete commit message
		rawMessage, err := QueryHTML(doc.Root.NodeID, ".MetadataMessage")

		if err != nil {
			return err
		}

		// Getting Commit Message and Contributors
		commitMessage, newAuthors, newReviewers := parseMessage(rawMessage)

		// Storing Contributors
		authors = append(authors, newAuthors...)
		reviewers = append(reviewers, newReviewers...)

		// Writing Commit Message
		textFile := parameters.commitsDir + "./Commits" + commitCode[0:6] + ".txt"
		err = WriteMessage(textFile, commitMessage)

		if err != nil {
			return err
		}

		// Getting next commit code
		search := strings.ReplaceAll(parameters.URL, "https://chromium.googlesource.com", "")

		html, err = QueryHTML(doc.Root.NodeID, "a[href*='"+search+commitCode+"%5E']")

		if err != nil {
			return err
		}

		commitCode = strings.Split(strings.TrimRight(html, "</a>"), ">")[1]
		// fmt.Println("Commit Code ", commitCode)

	}

	// Sort Authors and Reviewers
	sort.Strings(authors)
	sort.Strings(reviewers)

	// Merging and Writing contributors in CSV
	Merge.MergeWrite(authors, reviewers, parameters.contributorDir)

	return nil
}

// Helper Functions

func WriteMessage(fileName string, commitMessage []string) error {
	f, err := os.Create(fileName)

	if err != nil {
		return err
	}

	defer f.Close()
	for _, message := range commitMessage {
		f.WriteString(message)
		f.WriteString("\n")
	}

	return nil
}

func QuerySelectorAll(NodeID dom.NodeID, Selector string) (*dom.QuerySelectorAllReply, error) {
	QueryNodes, err := c.DOM.QuerySelectorAll(ctx, &dom.QuerySelectorAllArgs{
		NodeID:   NodeID,
		Selector: Selector,
	})
	return QueryNodes, err
}

func QuerySelector(NodeID dom.NodeID, Selector string) (*dom.QuerySelectorReply, error) {
	QueryNode, err := c.DOM.QuerySelector(ctx, &dom.QuerySelectorArgs{
		NodeID:   NodeID,
		Selector: Selector,
	})
	return QueryNode, err
}

func GetOuterHTML(NodeId dom.NodeID) (*dom.GetOuterHTMLReply, error) {
	result, err := c.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
		NodeID: &NodeId,
	})
	return result, err
}

// Function to select a node and return html output for it
func QueryHTML(NodeID dom.NodeID, Selector string) (string, error) {
	QueryNode, err := QuerySelector(NodeID, Selector)

	if err != nil {
		return "", err
	}

	result, err := GetOuterHTML(QueryNode.NodeID)

	return result.OuterHTML, err

}

func parseMessage(rawMessage string) ([]string, []string, []string) {
	// Convert HTML characters
	r := strings.NewReplacer("&lt;", "<", "&gt;", ">")

	completeMessage := strings.Split(r.Replace(rawMessage), "\n")

	var authors, reviewers, message []string
	complete := false
	for _, value := range completeMessage {
		if strings.Contains(value, "Tested-by:") && (value[0] == ' ' || value[0] == 'T') {
			authors = append(authors, strings.TrimLeft(strings.Trim(value, " "), "Tested-by:")[1:])
		}

		if strings.Contains(value, "Reviewed-by:") && (value[0] == ' ' || value[0] == 'R') {
			reviewers = append(reviewers, strings.TrimLeft(strings.Trim(value, " "), "Reviewed-by:")[1:])
		}
		if !complete {
			if strings.Contains(value, "BUG=") {
				complete = true
			} else {
				message = append(message, value)
			}
		}
	}
	message[0] = strings.Split(message[0], ">")[1]

	return message, authors, reviewers
}
