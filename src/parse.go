package src

import (
	"context"
	"strings"

	"github.com/mafredri/cdp"
	"github.com/mafredri/cdp/protocol/dom"
)

// Returns Branch url from list of branch names given on repository page
func getBranchURL(ctx context.Context, domClient cdp.DOM, RootID dom.NodeID, branchName string) string {

	var branchURL string
	QueryNodes, err := domClient.QuerySelectorAll(ctx, &dom.QuerySelectorAllArgs{
		NodeID:   RootID,
		Selector: "ul.RefList-items > li > a",
	})
	isError(err)

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

// Pareses message and return commit message and contributors stat
func parseMessage(ctx context.Context, domClient cdp.DOM, NodeID dom.NodeID) ([]string, []string, []string) {

	// Get complete commit message
	rawMessage := InnerHTML(ctx, domClient, NodeID, ".MetadataMessage", "</pre>")
	// fmt.Println(rawMessage)
	r := strings.NewReplacer("&lt;", "<", "&gt;", ">")

	commitMessage := strings.Split(r.Replace(rawMessage), "\n")

	authors, reviewers := parseContributorStat(commitMessage)

	return commitMessage, authors, reviewers
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
