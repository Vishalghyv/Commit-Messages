# Commit-Messages


### Table of Contents
**[About](#about)**<br>
**[Installation Instructions](#installation-instructions)**<br>
**[Command Line Arguments](#command-line-arguments)**<br>
**[Milestone 1](#milestone-1)**<br>
**[Milestone 2](#milestone-2)**<br>
**[Milestone 3](#milestone-3)**<br>
**[Code Snippets](#code-snippets)**<br>

## About 
This Project implements scrapping of Git repositories on chromium for storing commit messages and information about contributors

## Installation Instructions
1. Launch chromium with remote-debugging-port=9222, more details [here](https://github.com/mafredri/cdp)
2. Run command `git clone https://github.com/Vishalghyv/Commit-Messages `
3. Run `cd Commit-Messages`
4. To execute the program run `go run main.go`

## Command Line Arguments

Command Line Arguments that can be passed
1. Commit Numbers by default `10`.  
  Can be specified by --commit-num
2. Repository URL by default `https://chromium.googlesource.com/chromiumos/platform/tast-tests/`  
  Can be specified by --url
3. Branch name by default `main`.  
  Can be specified by --branch
4. Timeout in sec --timeout by default `10` sec.  
  Can be specified by --timeout
5. Commit message files folder path by default `./Commits`.  
  Can be specified by --commits-dir
6. CSV files folder path. By default `./Commits`.  
  Can be specified by --csv-dir

## Milestone 1
### Last 10 commit messages in tast-tests repository

<img src="https://github.com/Vishalghyv/TastTests-Messages/blob/main/Screenshots/MileStone1.jpg" height="350" width="700" alt="MileStone 1">

### Process to solve

1. Navigate to the given URL

2. Parse the page to get branch URL

3. Parse the branch page to get commit message and next commit code

4. Store the commit message in a new file with format Commit + first six char of commit code

Navigate to next commit code


## Milestone 2
### Add command line arguments to your program

<img src="https://github.com/Vishalghyv/TastTests-Messages/blob/main/Screenshots/MileStone2.jpg" height="350" width="700" alt="MileStone 2">

### Process to solve

1. Using flags package adding command line arguments

2. Making repository url, branch name, number of commit messages, file path of saving commit message, timeout dynamic

3. And there values taken from command line arguments

## Milestone 3
### Parse commit messages

<img src="https://github.com/Vishalghyv/TastTests-Messages/blob/main/Screenshots/MileStone3.jpg" height="350" width="700" alt="MileStone 3">

### Process to solve

1. Parse commit messages to get author and reviewers

2. Store them in seprate slice

   Sort slice in end

3. Using merge sort merge algorithim

4. Merge the two with keeping the tab of last contributor so as to add there total contributions

5. Write each contributors contribution in csv file

6. Add the file path for csv file as command line argument as done in milestone 2


## Code Snippets

### Main function to parse the HTML

```
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
```
### Function to Merge Contributors

```
   // Collects contributors stats from author and reviewers slices
   func mergeContributors(authors []string, reviewers []string) map[string]Contributor {

	contributors := make(map[string]Contributor)

	for i := 0; i < len(authors); i++ {
		contributors[authors[i]] = Contributor{created: contributors[authors[i]].created + 1, reviewed: contributors[authors[i]].reviewed}
	}
	for i := 0; i < len(reviewers); i++ {
		contributors[reviewers[i]] = Contributor{created: contributors[reviewers[i]].created, reviewed: contributors[reviewers[i]].reviewed + 1}
	}

	return contributors
  }
```
