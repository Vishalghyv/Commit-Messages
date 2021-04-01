package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/dom"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/rpcc"
)

type Parameters struct {
	URL, branch, commitsDir, contributorDir string
	commitNum                               int
	timeout                                 time.Duration
}

type Contributor struct {
	name     string
	created  int
	reviewed int
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

	// var para = Parameters{commitNum: commitNum, URL: url, branch: branch, timeout: timeout, commitsDir: commitsDir, contributorDir: contributorDir}

	err := run(para)
	if err != nil {
		fmt.Println(err)
	}
}

var ctx context.Context
var c *cdp.Client

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

	// Connection
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
	fmt.Println(QueryNodes)

	if err != nil {
		return err
	}
	var branch string

	for _, nodeId := range QueryNodes.NodeIDs {

		result, err := GetOuterHTML(nodeId)

		if err != nil {
			return err
		}

		if strings.Contains(result.OuterHTML, "main") {
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

	// Get commit code for first message
	html, err := QueryHTML(doc.Root.NodeID, ".u-monospace.Metadata td")

	if err != nil {
		return err
	}

	commitCode := strings.Replace(strings.TrimRight(html, "</td>"), "<td>", "", 1)
	fmt.Println("Commit Code ", commitCode)

	// Store parsed contributors
	var authors []string
	var reviewers []string

	addition := ""
	if parameters.URL[len(parameters.URL)-1] == '/' {
		addition += "+/"
	} else {
		addition += "/+/"
	}

	// MileStone 1 - storing of commit messages
	for i := 1; i <= parameters.commitNum; i++ {

		Link := parameters.URL + addition + commitCode

		// Navigate to commit code url
		doc, err = OpenDoc(Link, "https://google.com")
		if err != nil {
			return err
		}
		// Get complete commit message
		html, err = QueryHTML(doc.Root.NodeID, ".MetadataMessage")

		if err != nil {
			return err
		}
		// Convert HTML characters
		r := strings.NewReplacer("&lt;", "<", "&gt;", ">")

		completeMessage := strings.Split(r.Replace(html), "\n")

		newAuthors, newReviewers, message := parseMessage(completeMessage)

		authors = append(authors, newAuthors...)
		reviewers = append(reviewers, newReviewers...)

		message[0] = strings.Split(message[0], ">")[1]

		// Store commit message in file
		if _, err := os.Stat(parameters.commitsDir); os.IsNotExist(err) {
			os.Mkdir(parameters.commitsDir, os.ModeDir|0755)
		}

		textFile := parameters.commitsDir + "./Commits" + commitCode[0:6] + ".txt"
		err = CreateFile(textFile, message)

		if err != nil {
			return err
		}

		search := strings.ReplaceAll(parameters.URL, "https://chromium.googlesource.com", "")
		search += addition

		// Get next commit code
		html, err = QueryHTML(doc.Root.NodeID, "a[href*='"+search+commitCode+"%5E']")

		if err != nil {
			return err
		}

		commitCode = strings.Split(strings.TrimRight(html, "</a>"), ">")[1]
		// fmt.Println("Commit Code ", commitCode)

	}

	// MileStone Three - Saving contributors data
	// Sort Authors and Reviewers
	sort.Strings(authors)
	sort.Strings(reviewers)

	var last Contributor

	if authors[0] > reviewers[0] {
		last.name = reviewers[0]
	} else {
		last.name = authors[0]
	}

	// Create contributor csv file
	if _, err := os.Stat(parameters.contributorDir); os.IsNotExist(err) {
		os.Mkdir(parameters.contributorDir, os.ModeDir|0755)
	}

	file, err := os.Create(parameters.contributorDir + "./Contributors.csv")

	if err != nil {
		return err
	}

	defer file.Close()

	writer := csv.NewWriter(file)

	defer writer.Flush()

	writer.Write([]string{
		"contributor",
		"created",
		"reviewed",
	})

	MergeSort(authors, reviewers, last, writer)

	return nil
}

func write(last Contributor, writer *csv.Writer) {
	if last.name == "" {
		return
	}
	fmt.Println("Writing", last.name, last.created, last.reviewed)
	writer.Write([]string{
		last.name,
		strconv.Itoa(last.created),
		strconv.Itoa(last.reviewed),
	})
}

func MergeSort(authors []string, reviewers []string, last Contributor, writer *csv.Writer) {
	var i, j int
	// Merge Sorted authors and reviewers, and Write unqiue contributors in csv file
	for i < len(authors) && j < len(reviewers) {
		if authors[i] < reviewers[j] {

			if last.name != authors[i] {
				// Write the contributor and create new contributor
				write(last, writer)
				last = Contributor{}
				last.name = authors[i]
			}
			last.created++
			i++
		} else if authors[i] > reviewers[j] {

			if last.name != reviewers[j] {
				// Write the contributor and create new contributor
				write(last, writer)
				last = Contributor{}
				last.name = reviewers[j]
			}
			last.reviewed++
			j++
		} else {
			if last.name != authors[i] {
				// Write the contributor and create new contributor
				write(last, writer)
				last = Contributor{}
				last.name = authors[i]
			}
			last.created++
			last.reviewed++
			i++
			j++
		}

	}
	write(last, writer)
	last = Contributor{}

	for i < len(authors) {
		if i == len(authors)-1 && last.name == authors[i] {
			last.created++
			write(last, writer)
		}
		if last.name != authors[i] {
			write(last, writer)
			last = Contributor{}
			last.name = authors[i]
		}
		last.created++
		i++
	}

	for j < len(reviewers) {
		if j == len(reviewers)-1 && last.name == reviewers[j] {
			last.reviewed++
			write(last, writer)
		}

		if last.name != reviewers[j] {
			write(last, writer)
			last = Contributor{}
			last.name = reviewers[j]
		}
		last.reviewed++
		j++
	}
}

func CreateFile(fileName string, commitMessage []string) error {
	f, err := os.Create(fileName)

	if err != nil {
		return err
	}

	defer f.Close()
	for _, message := range commitMessage {
		_, err2 := f.WriteString(message)
		f.WriteString("\n")

		if err2 != nil {
			return err2
		}
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

func parseMessage(completeMessage []string) ([]string, []string, []string) {
	var authors, reviewers, message []string
	complete := false
	for _, value := range completeMessage {
		if strings.Contains(value, "Tested-by:") && (value[0] == ' ' || value[0] == 'T') {
			// fmt.Println(strings.TrimLeft(strings.Trim(value, " "), "Tested-by:")[1:])
			// fmt.Println(value)
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
	return authors, reviewers, message
}
