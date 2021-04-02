# Commit-Messages


### Table of Contents
**[About](#about)**<br>
**[Installation Instructions](#installation-instructions)**<br>
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

## Milestone 1
### Last 10 commit messages in tast-tests repository

<img src="https://github.com/Vishalghyv/TastTests-Messages/blob/main/MileStone1.jpg" height="350" width="700" alt="MileStone 1">

### Process to solve

1. Navigate to the given URL

2. Parse the page to get branch URL

3. Parse the branch page to get commit message and next commit code

4. Store the commit message in a new file with format Commit + first six char of commit code

Navigate to next commit code


## Milestone 2
### Add command line arguments to your program

<img src="https://github.com/Vishalghyv/TastTests-Messages/blob/main/MileStone2.jpg" height="350" width="700" alt="MileStone 2">

### Process to solve

1. Using flags package adding command line arguments

2. Making repository url, branch name, number of commit messages, file path of saving commit message, timeout dynamic

3. And there values taken from command line arguments

## Milestone 3
### Parse commit messages

<img src="https://github.com/Vishalghyv/TastTests-Messages/blob/main/MileStone3.jpg" height="350" width="700" alt="MileStone 3">

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
### Algorithim to Merge and Write Contributors data in CSV file

```
   type Contributor struct {
      name string
      created int 
      reviewed int
   }
   
   -> sorted authors and reviewrs slice
   -> last of type Contributor intialized with first smallest name
  
   for i < len(authors) && j < len(reviewers) {
		if (authors[i] < reviewers[j]) {
			Insert author[i]
			i++
		} else if (authors[i] > reviewers[j]) {
			Insert reviewer[i]
			j++
		} else {
			Increase both created and reviewed count
			i++
			j++
		}
	} 

	for i < len(authors) {
		Insert left authors
		i++
	}

	for j < len(reviewers) {
		Insert left reviewers
		j++
	}
```
