package src

import (
	"context"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/protocol/dom"
)

// Returns document root NodeID
func GetRootNodeID(ctx context.Context, domClient cdp.DOM) dom.NodeID {
	doc, err := domClient.GetDocument(ctx, nil)
	isError(err)

	return doc.Root.NodeID
}

// Invokes QuerySelectorAll from cdp DOM
func QuerySelectorAll(ctx context.Context, domClient cdp.DOM, NodeID dom.NodeID, Selector string) *dom.QuerySelectorAllReply {
	QueryNodes, err := domClient.QuerySelectorAll(ctx, &dom.QuerySelectorAllArgs{
		NodeID:   NodeID,
		Selector: Selector,
	})
	isError(err)
	return QueryNodes
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
func QueryHTML(ctx context.Context, domClient cdp.DOM, NodeID dom.NodeID, Selector string) string {
	QueryNode := QuerySelector(ctx, domClient, NodeID, Selector)

	result := GetOuterHTML(ctx, domClient, QueryNode.NodeID)

	return result.OuterHTML
}
