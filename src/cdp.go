package src

import (
	"context"
	"strings"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/protocol/dom"
)

// Returns document root NodeID
func GetRootNodeID(ctx context.Context, domClient cdp.DOM) dom.NodeID {
	doc, err := domClient.GetDocument(ctx, nil)
	isError(err)

	return doc.Root.NodeID
}

// Invokes QuerySelector from cdp DOM
func QuerySelector(ctx context.Context, domClient cdp.DOM, NodeID dom.NodeID, Selector string) *dom.QuerySelectorReply {
	QueryNode, err := domClient.QuerySelector(ctx, &dom.QuerySelectorArgs{
		NodeID:   NodeID,
		Selector: Selector,
	})
	isError(err)
	return QueryNode
}

// Invokes GetOuterHTMl from cdp DOM
func GetOuterHTML(ctx context.Context, domClient cdp.DOM, NodeId dom.NodeID) *dom.GetOuterHTMLReply {
	result, err := domClient.GetOuterHTML(ctx, &dom.GetOuterHTMLArgs{
		NodeID: &NodeId,
	})
	isError(err)
	return result
}

// Selects a node using selector and return html output for it
func InnerHTML(ctx context.Context, domClient cdp.DOM, NodeID dom.NodeID, Selector string, ClosingTag string) string {
	QueryNode := QuerySelector(ctx, domClient, NodeID, Selector)

	result := GetOuterHTML(ctx, domClient, QueryNode.NodeID)
	innerHTML := strings.SplitN(strings.TrimRight(result.OuterHTML, ClosingTag), ">", 2)[1]

	return innerHTML
}
