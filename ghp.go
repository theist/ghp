package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v32/github"
	"github.com/joho/godotenv"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/net/context"
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

// read stored token from disk file
func readStoredToken() string {
	homeDir, err := homedir.Dir()
	if err != nil {
		log.Printf("Error getting homedir %v", err)
		return ""
	}
	file, err := os.Open(filepath.Join(homeDir, USER_TOKEN))
	if err != nil {
		log.Printf("Error opening file %v: %v", filepath.Join(homeDir, USER_TOKEN), err)
		return ""
	}
	defer file.Close()
	bytes, err := ioutil.ReadAll(file)
	filecontent := string(bytes)

	return strings.TrimSpace(filecontent)
}

// write a token to file
func writeAuthToken(token string) error {
	homeDir, err := homedir.Dir()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(homeDir, USER_TOKEN), []byte(token), 0600)
	if err != nil {
		return err
	}
	return nil
}

func validToken(token string) bool {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if token == "" {
		log.Print("Empty token")
		return false
	}
	log.Printf("trying token: %v", token)
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

func main() {
	homeDir, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
	}
	godotenv.Load(".env", filepath.Join(homeDir, USER_CONFIG), GLOBAL_CONFIG)
	authToken := readStoredToken()
	if !validToken(authToken) { // do oauth authorization

		res, err := oauthCreateDeviceRequest()
		if err != nil {
			log.Fatal("Error creating device request: ", err)
		}

		token, err := getOauthToken(res)
		if err != nil {
			log.Fatal("Error getting oauth token: ", err)
		}
		writeAuthToken(*token)
		authToken = *token
	}
	log.Printf("Token: %v", authToken)
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: authToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	user, _, err := client.Users.Get(ctx, "")
	fmt.Printf("Authenticated as: %v\n", user.GetLogin())
	orgs, _, err := client.Organizations.List(ctx, "", nil)
	for _, org := range orgs {
		fmt.Printf(" - %v\n", org.GetLogin())
		projects, _, err := client.Organizations.ListProjects(ctx, org.GetLogin(), nil)
		for _, prj := range projects {
			fmt.Printf("   - %v\n", prj.GetName())
		}
		if err != nil {
			log.Fatal("Error")
		}
	}
}
