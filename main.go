package main

import (
	"context"
	"fmt"
	"time"
	"strings"
	"os"
	"flag"
	"sort"
	"encoding/csv"
	"strconv"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/dom"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/rpcc"
)

type Parameters struct {
	URL, branch, commitsDir, contributorDir string
	commitNum int
	timeout time.Duration
}

type Contributor struct {
	name string
	created int 
	reviewed int
}

func main() {
	var commitNum int
	var url, branch, commitsDir, contributorDir string
	var timeout time.Duration
	// Command Line Arguments
	flag.IntVar(&commitNum, "commit-num", 10, "number of last commit messages")
	flag.StringVar(&url, "url", "https://chromium.googlesource.com/chromiumos/platform/tast-tests/", "Repository URL")
	flag.StringVar(&branch, "branch", "main", "Branch Name")
	flag.DurationVar(&timeout, "timeout", 5 * time.Second, "Maximum time program to run")
	flag.StringVar(&commitsDir, "commits-dir", "./Commits/", "Commit message folder path")
	flag.StringVar(&contributorDir, "contributor-dir", "./Commits", "Contributor CSV folder path")
	flag.Parse()

	var para = Parameters{commitNum: commitNum, URL: url, branch: branch, timeout: timeout, commitsDir: commitsDir, contributorDir: contributorDir}

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

	// Connection
	// Initiate a new RPC connection to the Chrome DevTools Protocol target.
	conn, err := rpcc.DialContext(ctx, pt.WebSocketDebuggerURL)
	if err != nil {
		return err
	}
	defer conn.Close() // Leaving connections open will leak memory.

	c := cdp.NewClient(conn)

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
	OpenDoc := func (URL string, Referrer string) (*dom.GetDocumentReply, error) {
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
	result, err := c.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
		NodeID: &doc.Root.NodeID,
	})
	if err != nil {
		return err
	}

	// Perform search for the branch keyword
	searchId,err := c.DOM.PerformSearch(ctx, &dom.PerformSearchArgs{
		Query: parameters.branch,
	})
	if err != nil {
		return err
	}

	// Get Search Result
	nodeIds,err := c.DOM.GetSearchResults(ctx, &dom.GetSearchResultsArgs{
		SearchID: searchId.SearchID,
		FromIndex: 0,
		ToIndex: searchId.ResultCount,
	})
	if err != nil {
		return err
	}

	var branch string

	// Parse Search Results to get branch url
	for _, value := range nodeIds.NodeIDs {

		att, err := c.DOM.GetAttributes(ctx, &dom.GetAttributesArgs{
			NodeID: value,
		})

		// Check if Node Exists and have attributes containing href
		if err != nil || len(att.Attributes) == 0{
			continue;
		}

		for index, attributes := range att.Attributes {
			if attributes == "href" {
				branch = att.Attributes[index + 1]
			}
		}
		if (branch != "") {
			fmt.Println("Branch link", branch)
			break
		}
	  }

	// Navigate to branch
	Link ="https://chromium.googlesource.com" + branch

	doc, err = OpenDoc(Link, "https://google.com")
	if err != nil {
		return err
	}

	// Function to select a node and return html output for it
	QueryHTML := func (NodeID dom.NodeID, Selector string) (string, error) {
		QueryNode, err := c.DOM.QuerySelector(ctx, &dom.QuerySelectorArgs{
			NodeID: NodeID,
			Selector: Selector,
		})

		if err != nil {
			return "", err
		}

		result, err = c.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
			NodeID: &QueryNode.NodeID,
		})

		return result.OuterHTML, err

	}

	// Get commit code for first message
	html, err := QueryHTML(doc.Root.NodeID, ".u-monospace.Metadata td")

	if err != nil {
		return err
	}

	commitCode := strings.Replace(strings.TrimRight(html,"</td>"), "<td>", "", 1)
	fmt.Println("Commit Code ", commitCode)

	// Store parsed contributors
	var authors []string
	var reviewers []string

	addition := ""
	if parameters.URL[len(parameters.URL) -1] == '/' {
		addition += "+/"
	} else {
		addition += "/+/"
	}

	// MileStone 1 - storing of commit messages
	for i := 1; i <= parameters.commitNum; i++ {
		
		Link :=parameters.URL + addition + commitCode

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
		r := strings.NewReplacer("&lt;","<", "&gt;", ">")

		completeMessage := strings.Split(r.Replace(html), "\n")
		commitMessage := strings.Split(completeMessage[0], ">")[1]
		// fmt.Println("THis is hell", completeMessage)
		// Store contributor info
		for _, value := range completeMessage { 
			if strings.Contains(value, "Tested-by:")  && (value[0] == ' ' || value[0] == 'T'){
				// fmt.Println(strings.TrimLeft(strings.Trim(value, " "), "Tested-by:")[1:])
				// fmt.Println(value)
				authors = append(authors, strings.TrimLeft(strings.Trim(value, " "), "Tested-by:")[1:])
			}

			if strings.Contains(value, "Reviewed-by:") && (value[0] == ' ' || value[0] == 'R') {
				reviewers = append(reviewers, strings.TrimLeft(strings.Trim(value, " "), "Reviewed-by:")[1:])
			}
		} 
		
		// Store commit message in file
		if _, err := os.Stat(parameters.commitsDir); os.IsNotExist(err) {
			os.Mkdir(parameters.commitsDir, os.ModeDir|0755)
		}
	
		textFile := parameters.commitsDir + "./Commits" + commitCode[0:6] + ".txt"
		err = CreateFile(textFile, commitMessage)

		if err != nil {
			return err
		}

		search := strings.ReplaceAll(parameters.URL, "https://chromium.googlesource.com", "")
		search += addition

		// Get next commit code
		html, err = QueryHTML(doc.Root.NodeID, "a[href*='" + search + commitCode + "%5E']")

		if err != nil {
			return err
		}

		commitCode = strings.Split(strings.TrimRight(html,"</a>"), ">")[1]
		fmt.Println("Commit Code ", commitCode)

	}

	// MileStone Three - Saving contributors data
	// Sort Authors and Reviewers
	sort.Sort(sort.StringSlice(authors))
	sort.Sort(sort.StringSlice(reviewers))
	var i,j int

	var last Contributor

	if (authors[0] > reviewers[0]) {
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

	// Function to write contributors in csv
	write := func( last Contributor) {
		if (last.name == "") {
			return
		}
		fmt.Println("Writing", last.name, last.created, last.reviewed)
		writer.Write([]string{
			last.name,
			strconv.Itoa(last.created),
			strconv.Itoa(last.reviewed),
		})
	}

	// Merge Sorted authors and reviewers, and Write unqiue contributors in csv file
	for i < len(authors) && j < len(reviewers) {
		if (authors[i] < reviewers[j]) {
			
			if (last.name != authors[i]) {
				// Write the contributor and create new contributor
				write(last)
				last = Contributor{}
				last.name = authors[i]
			}
			last.created++
			i++
		} else if (authors[i] > reviewers[j]) {
			
			if (last.name != reviewers[j]) {
				// Write the contributor and create new contributor
				write(last)
				last = Contributor{}
				last.name = reviewers[j]
			}
			last.reviewed++
			j++
		} else {
			if (last.name != authors[i]) {
				// Write the contributor and create new contributor
				write(last)
				last = Contributor{}
				last.name = authors[i]
			}
			last.created++
			last.reviewed++
			i++
			j++
		}

	}
	write(last)
	last = Contributor{}

	for i < len(authors) {
		if (i == len(authors)-1 && last.name == authors[i]) {
			last.created++
			write(last)
		}
		if (last.name != authors[i]) {
			write(last)
			last = Contributor{}
			last.name = authors[i]
		}
		last.created++
		i++
	}

	for j < len(reviewers) {
		if (j == len(reviewers)-1 && last.name == reviewers[j]) {
			last.reviewed++
			write(last)
		}

		if (last.name != reviewers[j]) {
			write(last)
			last = Contributor{}
			last.name = reviewers[j]
		}
		last.reviewed++
		j++
	}


	return nil
}


func CreateFile(fileName string, commitMessage string) error {
	f, err := os.Create(fileName)

    if err != nil {
        return err
    }

    defer f.Close()

    _, err2 := f.WriteString(commitMessage)

    if err2 != nil {
        return err2
    }
	return nil

}