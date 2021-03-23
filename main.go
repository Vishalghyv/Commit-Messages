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
	URL, branch, folder, file string
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
	var url, branch, folder, file string
	var timeout time.Duration
	flag.IntVar(&commitNum, "commitNum", 10, "the count of items")
	flag.StringVar(&url, "url", "https://chromium.googlesource.com/chromiumos/platform/tast-tests/", "Repository URL")
	flag.StringVar(&branch, "branch", "main", "Branch Name")
	flag.DurationVar(&timeout, "timeout", 5 * time.Second, "Maximum time program to run")
	flag.StringVar(&folder, "folder", "./Commits/", "Folder Path")
	flag.StringVar(&file, "file", "./Commits", "file Path")
	flag.Parse()

	var para = Parameters{commitNum: commitNum, URL: url, branch: branch, timeout: timeout, folder: folder, file: file}

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

	// Navigating to main site
	
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

	
	Link := parameters.URL
	doc, err := OpenDoc(Link, "https://google.com")
	if err != nil {
		return err
	}

	// Get link for main branch

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

	nodeIds,err := c.DOM.GetSearchResults(ctx, &dom.GetSearchResultsArgs{
		SearchID: searchId.SearchID,
		FromIndex: 0,
		ToIndex: searchId.ResultCount,
	})
	if err != nil {
		return err
	}

	var branch string

	// Parse Search Results to get branch link
	for _, value := range nodeIds.NodeIDs {

		att, err := c.DOM.GetAttributes(ctx, &dom.GetAttributesArgs{
			NodeID: value,
		})

		// Node Exsists and have attributes containing href
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

	Link ="https://chromium.googlesource.com" + branch

	doc, err = OpenDoc(Link, "https://google.com")
	if err != nil {
		return err
	}

	// Function to select to select a node and return html output for it
	QueryHTML := func (NodeID dom.NodeID, Selector string) (string, error) {
		message, err := c.DOM.QuerySelector(ctx, &dom.QuerySelectorArgs{
			NodeID: NodeID,
			Selector: Selector,
		})

		if err != nil {
			return "", err
		}

		result, err = c.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
			NodeID: &message.NodeID,
		})

		return result.OuterHTML, err

	}

	html, err := QueryHTML(doc.Root.NodeID, ".u-monospace.Metadata td")

	if err != nil {
		return err
	}

	next := strings.TrimLeft(strings.TrimRight(html,"</td>"), "<td>")
	fmt.Println("Next ", next)

	//From Next Commit
	var authors []string
	var reviewers []string

	for i := 1; i <= parameters.commitNum; i++ {
		Link :="https://chromium.googlesource.com/chromiumos/platform/tast-tests/+/" + next

		doc, err = OpenDoc(Link, "https://google.com")
		if err != nil {
			return err
		}

		html, err = QueryHTML(doc.Root.NodeID, ".MetadataMessage")

		if err != nil {
			return err
		}
		// Convert HTML characters
		r := strings.NewReplacer("&lt;","<", "&gt;", ">")

		completeMessage := strings.Split(r.Replace(html), "\n")
		message := strings.Split(completeMessage[0], ">")[1]
		// fmt.Println("Commit Message:\n", completeMessage)

		for _, value := range completeMessage { 
			if strings.Contains(value, "Tested-by:") {
				authors = append(authors, strings.TrimLeft(value, "Tested-by:")[1:])
			}

			if strings.Contains(value, "Reviewed-by:") {
				reviewers = append(reviewers, strings.TrimLeft(value, "Reviewed-by:")[1:])
			}
		} 

		html, err = QueryHTML(doc.Root.NodeID, "a[href*='/chromiumos/platform/tast-tests/+/" + next + "%5E']")

		if err != nil {
			return err
		}
		
		textFile := parameters.folder + "./Commits" + next[0:6] + ".txt"
		err = CreateFile(textFile, message)

		if err != nil {
			return err
		}

		next = strings.Split(strings.TrimRight(html,"</a>"), ">")[1]
		fmt.Println("Next ", next)

	}

	// MileStone Three
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

	file, err := os.Create(parameters.file + "./Contributors.csv")

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

	write := func( last Contributor) {
		writer.Write([]string{
			last.name,
			strconv.Itoa(last.created),
			strconv.Itoa(last.reviewed),
		})
	}


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
				last.name = authors[j]
			}
			last.created++
			last.reviewed++
			i++
			j++
		}
	} 

	for i < len(authors) {
		// Write the contributor and create new contributor
		last.name = authors[i]
		last.created = 1
		last.reviewed = 0
		write(last)
		i++
	}

	for j < len(reviewers) {
		// Write the contributor and create new contributor
		last.name = reviewers[j]
		last.created = 0
		last.reviewed = 1
		write(last)
		j++
	}


	return nil
}


func CreateFile(fileName string, message string) error {
	f, err := os.Create(fileName)

    if err != nil {
        return err
    }

    defer f.Close()

    _, err2 := f.WriteString(message)

    if err2 != nil {
        return err2
    }
	return nil

}