/*
	Package main implements scrapping of Git repositories on chromium for
	storing commit messages and information about contributors
*/
package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/Vishalghyv/Commit-Messages/src"
)

func main() {
	var para = src.Parameters{}

	// Command Line Arguments
	flag.IntVar(&para.CommitNum, "commit-num", 10, "number of last commit messages")
	flag.StringVar(&para.URL, "url", "https://chromium.googlesource.com/chromiumos/platform/tast-tests/", "Repository URL")
	flag.StringVar(&para.Branch, "branch", "main", "Branch Name")
	flag.DurationVar(&para.Timeout, "timeout", 10*time.Second, "Maximum time program to run")
	flag.StringVar(&para.CommitsDir, "commits-dir", "./Commits/", "Commit message folder path")
	flag.StringVar(&para.ContributorDir, "csv-dir", "./Commits", "Contributor CSV folder path")
	flag.Parse()

	err := src.Run(para)
	if err != nil {
		fmt.Println(err)
	}
}
