package main

import (
	"context"
	"fmt"
	"log"
	"time"
	"reflect"
	"strings"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/devtool"
	"github.com/mafredri/cdp/protocol/dom"
	"github.com/mafredri/cdp/protocol/page"
	"github.com/mafredri/cdp/rpcc"
)

func main() {
	err := run(5 * time.Second)
	if err != nil {
		log.Fatal(err)
	}
}

func run(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Use the DevTools HTTP/JSON API to manage targets (e.g. pages, webworkers).
	devt := devtool.New("http://127.0.0.1:9221")
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

	// Create the Navigate arguments with the optional Referrer field set.
	navArgs := page.NewNavigateArgs("https://chromium.googlesource.com/chromiumos/platform/tast-tests/").
		SetReferrer("https://google.com")
	nav, err := c.Page.Navigate(ctx, navArgs)
	if err != nil {
		return err
	}

	// Wait until we have a DOMContentEventFired event.
	if _, err = domContent.Recv(); err != nil {
		return err
	}

	fmt.Printf("Page loaded with frame ID: %s\n", nav.FrameID)

	// Fetch the document root node. We can pass nil here
	// since this method only takes optional arguments.
	doc, err := c.DOM.GetDocument(ctx, nil)
	if err != nil {
		return err
	}

	// Get the outer HTML for the page.
	result, err := c.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
		NodeID: &doc.Root.NodeID,
	})
	if err != nil {
		return err
	}

	if result == nil {
		return err
	}

	// fmt.Printf("HTML: %s\n", result.OuterHTML)
	// result, err = c.DOM.GetElementsByTagName('a')

	searchId, resultCount := c.DOM.PerformSearch(ctx, &dom.PerformSearchArgs{
		Query: "main",
	})

	
	fmt.Println("var1 = ", reflect.TypeOf(searchId.SearchID)) 
    fmt.Println("var2 = ", reflect.TypeOf(resultCount)) 

	nodeIds, nodeerr := c.DOM.GetSearchResults(ctx, &dom.GetSearchResultsArgs{
		SearchID: searchId.SearchID,
		FromIndex: 0,
		ToIndex: searchId.ResultCount,
	})

	fmt.Println("var1 = ", reflect.TypeOf(nodeerr))

	for _, value := range nodeIds.NodeIDs {
		// fmt.Printf("- %d\n", value)
		result, err := c.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
			NodeID: &value,
		})
		if err != nil {
			fmt.Printf(" ERrr : %s\n", err)
		}
	
		fmt.Printf("HTML: %s\n", result.OuterHTML)
		att, err := c.DOM.GetAttributes(ctx, &dom.GetAttributesArgs{
			NodeID: value,
		})

		if err != nil {
			fmt.Printf(" ERrr : %s\n", err)
		}
		if len(att.Attributes) == 0 {
			continue;
		}

		fmt.Println("HTML: ", att.Attributes)
		fmt.Printf("HTML: %s", att.Attributes[len(att.Attributes)-1])
		// fmt.Println("var1 = ", reflect.TypeOf(att.Attributes)) 
	  }
	  

	  navArgs = page.NewNavigateArgs("https://chromium.googlesource.com/chromiumos/platform/tast-tests/+/refs/heads/main").
	  SetReferrer("https://google.com")

	  nav, err = c.Page.Navigate(ctx, navArgs)
	if err != nil {
		return err
	}

	// Wait until we have a DOMContentEventFired event.
	if _, err = domContent.Recv(); err != nil {
		return err
	}

	fmt.Printf("Page loaded with frame ID: %s\n", nav.FrameID)


	doc, err = c.DOM.GetDocument(ctx, nil)
	if err != nil {
		return err
	}

	// Get the outer HTML for the page.
	message, err := c.DOM.QuerySelector(ctx, &dom.QuerySelectorArgs{
		NodeID: doc.Root.NodeID,
		Selector: ".MetadataMessage",
	})
	if err != nil {
		return err
	}


	fmt.Printf("HTML: %d\n", message.NodeID)

	result, err = c.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
		NodeID: &message.NodeID,
	})
	if err != nil {
		fmt.Printf(" ERrr : %s\n", err)
	}

	fmt.Printf("HTML: %s\n", strings.Split(strings.Split(result.OuterHTML, "\n")[0], ">")[1])

	message, err = c.DOM.QuerySelector(ctx, &dom.QuerySelectorArgs{
		NodeID: doc.Root.NodeID,
		Selector: "a[href*='/chromiumos/platform/tast-tests/+/refs/heads/main%5E']",
	})
	if err != nil {
		return err
	}


	fmt.Printf("HTML: %d\n", message.NodeID)

	result, err = c.DOM.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
		NodeID: &message.NodeID,
	})
	if err != nil {
		fmt.Printf(" ERrr : %s\n", err)
	}

	fmt.Printf("HTML: %s\n", result.OuterHTML)

	// Saving

	// // Capture a screenshot of the current page.
	// screenshotName := "screenshot.jpg"
	// screenshotArgs := page.NewCaptureScreenshotArgs().
	// 	SetFormat("jpeg").
	// 	SetQuality(80)
	// screenshot, err := c.Page.CaptureScreenshot(ctx, screenshotArgs)
	// if err != nil {
	// 	return err
	// }
	// if err = ioutil.WriteFile(screenshotName, screenshot.Data, 0644); err != nil {
	// 	return err
	// }

	// fmt.Printf("Saved screenshot: %s\n", screenshotName)

	// pdfName := "page.pdf"
	// f, err := os.Create(pdfName)
	// if err != nil {
	// 	return err
	// }
	// //fmt.Printf("Create file: %s\n", screenshotName)

	// pdfArgs := page.NewPrintToPDFArgs().
	// 	SetTransferMode("ReturnAsStream") // Request stream.
	// pdfData, err := c.Page.PrintToPDF(ctx, pdfArgs)
	// if err != nil {
	// 	return err
	// }

	// sr := c.NewIOStreamReader(ctx, *pdfData.Stream)
	// r := bufio.NewReader(sr)

	// // Write to file in ~r.Size() chunks.
	// _, err = r.WriteTo(f)
	// if err != nil {
	// 	return err
	// }

	// err = f.Close()
	// if err != nil {
	// 	return err
	// }

	// fmt.Printf("Saved PDF: %s\n", pdfName)

	return nil
}