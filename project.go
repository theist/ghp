package main

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/go-github/v32/github"
	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2"
)

type card interface {
	getURL() string
	toListString() string
	match(filters [][]string) bool
}

type issue struct {
	url        string
	createdAt  github.Timestamp
	ghIssue    *github.Issue
	labels     []*github.Label
	repository *github.Repository
}

func (i issue) getURL() string {
	return i.url
}

func (i issue) toListString() string {
	res := "issue: "
	res += fmt.Sprintf("%v#%v", i.repository.GetName(), i.ghIssue.GetNumber())
	if i.ghIssue.GetState() == "closed" {
		res += "(closed)"
	}
	res += " " + i.ghIssue.GetTitle() + " "
	assignee := i.ghIssue.GetAssignee()
	if assignee != nil {
		res += "@" + assignee.GetLogin() + " "
	}
	labels := i.ghIssue.Labels
	if len(labels) != 0 {
		labelnames := make([]string, 0, len(labels))
		for _, label := range labels {
			labelnames = append(labelnames, label.GetName())
		}
		res += strings.Join(labelnames, ",")
	}
	return res
}

func (i issue) match(filters [][]string) bool {
	if len(filters) == 0 {
		return true
	}
	issueString := "issue " + i.toListString()
	if i.ghIssue.Assignee == nil {
		issueString += " unassigned"
	}
	for _, orFilter := range filters {
		subRes := true
		for _, andFilter := range orFilter {
			if !strings.Contains(issueString, andFilter) { // if AND subfilter fails one break and set false
				subRes = false
				break
			}
		}
		if subRes {
			return true // if only one OR filter is true return inmediately true
		}
	}
	return false
}

func (i *issue) labelString() string {
	res := ""
	labels := i.ghIssue.Labels
	if len(labels) != 0 {
		labelnames := make([]string, 0, len(labels))
		for _, label := range labels {
			labelnames = append(labelnames, label.GetName())
		}
		res += strings.Join(labelnames, ",")
	}
	return res
}

func (i *issue) lenLabelString() int {
	return utf8.RuneCountInString(i.labelString())
}

type note struct {
	url       string
	text      string
	createdAt github.Timestamp
}

func (n note) getURL() string {
	return n.url
}

func (n note) toListString() string {
	return "note: " + n.text
}

// uses "note " +  note test for matching
func (n note) match(filters [][]string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, orFilter := range filters {
		subRes := true
		for _, andFilter := range orFilter {
			if !strings.Contains("note "+n.text, andFilter) { // if AND subfilter fails one break and set false
				subRes = false
				break
			}
		}
		if subRes {
			return true // if only one OR filter is true return inmediately true
		}
	}
	return false
}

type column struct {
	name  string
	url   string
	id    int64
	cards []card
}

func buildCard(p *ProjectProxy, c *github.ProjectCard) (card, error) {
	url := c.GetURL()
	noteText := c.GetNote()
	if noteText != "" {
		n := new(note)
		n.text = noteText
		n.url = url
		n.createdAt = c.GetCreatedAt()
		return n, nil // returns note
	}
	i, err := p.getIssueByURL(c.GetContentURL())
	if err != nil {
		return nil, err
	}
	i.url = url
	i.createdAt = c.GetCreatedAt()
	return i, nil // returns issue
}

func (c *column) pullCards(p *ProjectProxy) error {
	// log.Printf("pullig cards for %v", c.id)
	cards, res, err := p.client.Projects.ListProjectCards(*p.context, c.id, nil)
	if err != nil {
		return fmt.Errorf("Error Getting cards for %v: %v", c.id, err)
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("Error Getting cards for %v: http: %v", c.id, res.Status)
	}
	for _, card := range cards {
		if !card.GetArchived() {
			newCard, err := buildCard(p, card)
			// log.Printf("new Card: %+v", newCard)
			if err != nil {
				return fmt.Errorf("Error Getting card for %v: %v", c.id, err)
			}
			c.cards = append(c.cards, newCard)
		}
	}
	return nil
}

// ProjectProxy Class for interacting github's project
type ProjectProxy struct {
	client      *github.Client
	context     *context.Context
	authToken   string
	cache       *cache.Cache
	cacheHits   int
	cacheMisses int
	user        string
	columns     []column
}

func (p *ProjectProxy) requestAPI(url string, v interface{}) error {
	cached, found := p.cache.Get(url)
	if found {
		p.cacheHits++
		reflect.ValueOf(v).Elem().Set(reflect.ValueOf(cached).Elem()) // v = x with interfaces
		return nil
	}
	p.cacheMisses++
	req, _ := p.client.NewRequest("GET", url, nil)
	res, err := p.client.Do(*p.context, req, v)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("Error Getting %v: %v", url, res.Status)
	}
	p.cache.Add(url, v, cache.DefaultExpiration)
	// TODO: process API response
	return nil
}

func (p *ProjectProxy) getIssueByURL(url string) (*issue, error) {
	i := new(github.Issue)
	req, _ := p.client.NewRequest("GET", url, nil)
	_, err := p.client.Do(*p.context, req, i)
	if err != nil {
		return nil, err
	}
	pIssue := new(issue)
	pIssue.ghIssue = i
	repo := new(github.Repository)
	err = p.requestAPI(i.GetRepositoryURL(), repo)
	if err != nil {
		return nil, err
	}
	pIssue.repository = repo
	return pIssue, nil
}

func (p *ProjectProxy) pullColums(projectID int64) error {
	// log.Printf("Pull columns %v", projectID)
	cols, res, err := p.client.Projects.ListProjectColumns(*p.context, projectID, nil)
	if err != nil {
		return fmt.Errorf("Error Getting columns for %v: %v", projectID, err)
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("Error Getting columns for %v: http: %v", projectID, res.Status)
	}
	if len(cols) < 1 {
		return fmt.Errorf("Error Getting columns for %v: Zero items", projectID)
	}
	for _, c := range cols {
		col := new(column)
		col.name = c.GetName()
		col.id = c.GetID()
		col.url = c.GetURL()
		err := col.pullCards(p)
		if err != nil {
			return err
		}
		p.columns = append(p.columns, *col)
	}
	return nil
}

// Project Proxy initializer
func (p *ProjectProxy) init(state ghpConfig, projectID int64) error {
	p.cache = cache.New(10*time.Minute, 15*time.Minute)
	ctx := context.Background()
	p.authToken = state.AccessToken
	p.user = state.User
	p.context = &ctx
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: p.authToken},
	)
	tc := oauth2.NewClient(*p.context, ts)
	p.client = github.NewClient(tc)
	return nil
}

func (p *ProjectProxy) listProject(filter [][]string) {
	for _, col := range p.columns {
		fmt.Println(col.name + ":")
		for _, card := range col.cards {
			if card.match(filter) {
				fmt.Println("   " + card.toListString())
			}
		}
	}
}
