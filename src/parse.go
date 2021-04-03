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
