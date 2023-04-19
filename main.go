package main

import (
	"fmt"

	"ariga.io/gh-atlas/internal/github"
	"github.com/cli/go-gh"
)

func main() {
	fmt.Println("hi world, this is the gh-atlas extension!")
	client, err := gh.RESTClient(nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	response := struct{ Login string }{}
	err = client.Get("user", &response)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("running as %s\n", response.Login)
	if err = logic(); err != nil {
		fmt.Println("error github: ", err)
		return
	}
}

func logic() error {
	branchName := "branch-name"
	commitMsg := "commit-msg"
	prTitle := "pr-title"
	repo, err := github.NewGitHubRepository()
	if err != nil {
		return err
	}
	fmt.Println("ok1")
	if err = repo.SetAtlasToken(""); err != nil {
		return err
	}
	if err = repo.CheckoutNewBranch(branchName); err != nil {
		return err
	}
	if err = repo.AddAtlasYaml("path"); err != nil {
		return err
	}
	if err = repo.CommitChanges(commitMsg); err != nil {
		return err
	}
	if err = repo.PushChanges(branchName); err != nil {
		return err
	}
	return repo.CreatePR(prTitle, branchName)
}

// For more examples of using go-gh, see:
// https://github.com/cli/go-gh/blob/trunk/example_gh_test.go
