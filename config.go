package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/go-github/v32/github"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/oauth2"
)

type ghpConfig struct {
	AccessToken        string `json:"access_token"`
	User               string `json:"user"`
	DefaultProject     string `json:"default_project"`
	DefaultProjectID   int64  `json:"default_project_id"`
	DefaultProjectType string `json:"default_project_type"`
	Organization       string `json:"organization"`
}

// load Loads json state from disk
func stateLoad() (*ghpConfig, error) {
	st := ghpConfig{
		AccessToken:        "",
		User:               "",
		DefaultProject:     "",
		DefaultProjectType: "",
	}
	homeDir, err := homedir.Dir()
	if err != nil {
		return &st, err
	}

	file, err := os.Open(filepath.Join(homeDir, userState))
	if err != nil {
		return &st, err
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return &st, err
	}
	err = json.Unmarshal(bytes, &st)
	if err != nil {
		return &st, err
	}
	return &st, nil
}

func (state *ghpConfig) save() error {
	homeDir, err := homedir.Dir()
	if err != nil {
		return fmt.Errorf("error Saving state: %v", err)
	}
	if state == nil {
		return fmt.Errorf("can't save a nil state")
	}
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("error marshalling %v", data)
	}
	err = ioutil.WriteFile(filepath.Join(homeDir, userState), data, 0600)
	if err != nil {
		return fmt.Errorf("error Saving state: %v", err)
	}
	return nil
}

// TODO: Use central client
func renewConfig(state *ghpConfig) error {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: state.AccessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("error getting user: %v", err)
	}
	fmt.Printf("Authenticated as: %v\n", user.GetLogin())
	state.User = user.GetLogin()
	orgs, _, err := client.Organizations.List(ctx, "", nil)
	if err != nil {
		return fmt.Errorf("error getting orgs for user %v : %v", state.User, err)
	}
	if len(orgs) == 0 {
		return fmt.Errorf("no orgs for user %v", state.User)
	}
	orgnames := []string{}
	for _, org := range orgs {
		orgnames = append(orgnames, org.GetLogin())
	}
	index, err := choice("Select organization", orgnames)
	if err != nil {
		return err
	}
	state.Organization = orgnames[index]
	projects, _, err := client.Organizations.ListProjects(ctx, state.Organization, nil)
	if err != nil {
		return fmt.Errorf("error getting projects for org %v : %v", state.Organization, err)
	}
	if len(orgs) == 0 {
		return fmt.Errorf("no orgs for user %v", state.Organization)
	}
	projectList := []string{}
	projectIDs := []int64{}
	for _, prj := range projects {
		projectList = append(projectList, prj.GetName())
		projectIDs = append(projectIDs, prj.GetID())
	}
	projectIndex, err := choice("Select project", projectList)
	if err != nil {
		return err
	}
	state.DefaultProject = projectList[projectIndex]
	state.DefaultProjectID = projectIDs[projectIndex]
	state.DefaultProjectType = "organization"
	return nil
}
