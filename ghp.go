package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v32/github"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/oauth2"
)

type deviceOauthResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type oauthAuthCodeResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type ghpState struct {
	AccessToken        string `json:"access_token"`
	User               string `json:"user"`
	DefaultProject     string `json:"default_project"`
	DefaultProjectType string `json:"default_project_type"`
	Organization       string `json:"organization"`
}

// global var to hold state
var state ghpState

// load Loads json state from disk
func stateLoad() (ghpState, error) {
	st := ghpState{
		AccessToken:        "",
		User:               "",
		DefaultProject:     "",
		DefaultProjectType: "",
	}
	homeDir, err := homedir.Dir()
	if err != nil {
		return st, err
	}

	file, err := os.Open(filepath.Join(homeDir, userState))
	if err != nil {
		return st, err
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return st, err
	}
	err = json.Unmarshal(bytes, &st)
	if err != nil {
		return st, err
	}
	return st, nil
}

func (state *ghpState) save() {
	homeDir, err := homedir.Dir()
	if err != nil {
		log.Printf("Error Saving state: %v", err)
		return
	}
	if state == nil {
		log.Print("Can't save a nil state")
		return
	}
	data, err := json.Marshal(state)
	err = ioutil.WriteFile(filepath.Join(homeDir, userState), data, 0600)
	if err != nil {
		log.Printf("Error Saving state: %v", err)
		return
	}
}

func oauthCreateDeviceRequest() (*deviceOauthResponse, error) {
	body := strings.NewReader(`client_id=0412cc5fb93b10a59e50&scope=repo`)
	req, err := http.NewRequest("POST", "https://github.com/login/device/code", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var responseJSON deviceOauthResponse
	err = json.Unmarshal(responseBody, &responseJSON)
	if err != nil {
		return nil, err
	}
	return &responseJSON, nil
}

func getOauthToken(device *deviceOauthResponse) (*string, error) {
	fmt.Println("A browser window will open")
	fmt.Println("Please insert this code to authorize this client")
	fmt.Println(device.UserCode)
	openBrowser(device.VerificationURI)
	fmt.Println("Press enter when done!")
	fmt.Scanln() // wait for Enter Key

	body := strings.NewReader(fmt.Sprintf("client_id=0412cc5fb93b10a59e50&device_code=%s&grant_type=urn:ietf:params:oauth:grant-type:device_code", device.DeviceCode))

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var responseAuth oauthAuthCodeResponse
	err = json.Unmarshal(responseBody, &responseAuth)
	if err != nil {
		return nil, err
	}
	res := responseAuth.AccessToken
	return &res, nil
}

func validToken(token string) bool {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if token == "" {
		log.Print("Empty token")
		return false
	}
	if err != nil {
		log.Println("Error Creating request", err)
		return false
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", "token "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Error Sending request", err)
		return false
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error Reading body", err)
		return false
	}
	if resp.StatusCode != 200 {
		log.Printf("Client error: %v", resp.StatusCode)
		return false
	}
	return true
}

// TODO: Show default project
func showDefaultProject() {
	log.Fatal("Unimplemented")
}

func renewOAuth() {
	res, err := oauthCreateDeviceRequest()
	if err != nil {
		log.Fatal("Error creating device request: ", err)
	}

	token, err := getOauthToken(res)
	if err != nil {
		log.Fatal("Error getting oauth token: ", err)
	}
	state.AccessToken = *token
}

func renewConfig() error {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: state.AccessToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("Error getting user: %v", err)
	}
	fmt.Printf("Authenticated as: %v\n", user.GetLogin())
	state.User = user.GetLogin()
	orgs, _, err := client.Organizations.List(ctx, "", nil)
	if err != nil {
		return fmt.Errorf("Error getting orgs for user %v : %v", state.User, err)
	}
	if len(orgs) == 0 {
		return fmt.Errorf("No orgs for user %v", state.User)
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
		return fmt.Errorf("Error getting projects for org %v : %v", state.Organization, err)
	}
	if len(orgs) == 0 {
		return fmt.Errorf("No orgs for user %v", state.Organization)
	}
	projectList := []string{}
	for _, prj := range projects {
		projectList = append(projectList, prj.GetName())
	}
	projectIndex, err := choice("Select project", projectList)
	if err != nil {
		return err
	}
	state.DefaultProject = projectList[projectIndex]
	state.DefaultProjectType = "organization"
	return nil
}

// TODO: Show help
func doHelp() {
	log.Fatal("Unimplemented")
}

func main() {
	state, err := stateLoad()
	if err != nil {
		fmt.Printf("Empty state: %v\n", err)
	}

	if len(os.Args) < 2 {
		showDefaultProject()
	}

	command := os.Args[1]

	switch command {
	case "auth":
		if validToken(state.AccessToken) {
			if !askForConfirmation("There's already valid token are you sure") {
				os.Exit(0)
			}
		}
		renewOAuth()
		if validToken(state.AccessToken) {
			state.save()
			fmt.Printf("Auth changes will clear options, please run 'ghp config'")
		}
	case "config":
		if !validToken(state.AccessToken) {
			fmt.Printf("There's no valid oauth token, please run 'ghp auth'")
		}
		err = renewConfig()
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(0)
		}
		if validToken(state.AccessToken) {
			state.save()
		}
	case "help":
		doHelp()
	default:
		fmt.Printf("Unsupported command %v\n\n", command)
		doHelp()
	}
}
