package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
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

type filterFlags []string

func (f *filterFlags) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func (f *filterFlags) String() string {
	str := "(" + strings.Join(*f, ") OR (") + ")"
	str = strings.ReplaceAll(str, ",", " AND ")
	return str
}

func (f *filterFlags) toFilters() [][]string {
	filters := [][]string{}
	for _, filter := range *f {
		filters = append(filters, strings.Split(filter, ","))
	}
	return filters
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

func getOAuthToken() string {
	res, err := oauthCreateDeviceRequest()
	if err != nil {
		log.Fatal("Error creating device request: ", err)
	}

	token, err := getOauthToken(res)
	if err != nil {
		log.Fatal("Error getting oauth token: ", err)
	}
	return *token
}

// TODO: Show help
func doHelp() {
	log.Fatal("Unimplemented")
}

func doList(state ghpConfig, cache *appCache, f filterFlags) {
	fmt.Printf("Requesting full project %v, this can take some time\n", state.DefaultProject)
	p := new(ProjectProxy)
	err := p.init(state, cache, state.DefaultProjectID)
	if err != nil {
		fmt.Printf("Error creating client %v", err)
	}
	err = p.pullColums(state.DefaultProjectID)
	if err != nil {
		fmt.Printf("Error reading project %v", err)
	}
	// log.Printf("prj %+v", p)
	if len(f) != 0 {
		fmt.Printf("Appliying filters: %v\n", f.String())
	}
	//p.listProject(f.toFilters())
	fancyList(p, f.toFilters())
	fmt.Printf("\ncache performance:\nHits: %v\nMiss:%v\n", cache.cacheHits, cache.cacheMiss)
}

func main() {
	state, err := stateLoad()
	if err != nil {
		fmt.Printf("Empty state: %v\n", err)
	}
	cache := initCache()

	// parse flags
	var filters filterFlags
	flag.Var(&filters, "filter", "Issue filtering, use a comma separated for AND filter and several -filter paramenters for OR filter")
	flag.Parse()

	if len(flag.Args()) < 2 {
		if !validToken(state.AccessToken) {
			fmt.Println("You don't have configured ghp yet, run 'ghp auth' and 'ghp config'")
			os.Exit(1)
		}
		if state.DefaultProjectID == 0 {
			fmt.Println("You don't have configured ghp yet, run 'ghp config'")
			os.Exit(1)
		}
		doList(*state, cache, filters)
		os.Exit(0)
	}

	command := flag.Arg(1)

	switch command {
	case "auth":
		if validToken(state.AccessToken) {
			if !askForConfirmation("There's already valid token are you sure") {
				os.Exit(0)
			}
		}
		state.AccessToken = getOAuthToken()
		if validToken(state.AccessToken) {
			err := state.save()
			if err != nil {
				fmt.Printf("Error saving config: %v", err)
				os.Exit(1)
			}
			fmt.Printf("Auth changes will clear options, please run 'ghp config'\n")
		}
	case "config":
		if !validToken(state.AccessToken) {
			fmt.Printf("There's no valid oauth token, please run 'ghp auth'")
		}
		err = renewConfig(state)
		if err != nil {
			fmt.Printf("%v", err)
			os.Exit(0)
		}
		if validToken(state.AccessToken) {
			err := state.save()
			if err != nil {
				fmt.Printf("Error saving state: %v", err)
			}
		}
	case "help":
		doHelp()
	case "list":
		doList(*state, cache, filters)
	default:
		fmt.Printf("Unsupported command %v\n\n", command)
		doHelp()
	}
}
