package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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

type ghpClient struct {
	oauthToken string
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

func createClient(authToken string) *ghpClient {
	c := new(ghpClient)
	c.oauthToken = authToken
	return c
}

//lint:ignore U1000 uninpremented
func (c *ghpClient) getToken() string {
	return c.oauthToken
}

func (c *ghpClient) validToken() (bool, error) {
	if c.oauthToken == "" {
		log.Print("Empty token")
		return false, nil
	}
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return false, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", "token "+c.oauthToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("error Sending request: %v", err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("error reading body: %v", err)
	}
	if resp.StatusCode != 200 {
		return false, fmt.Errorf("client error: %v", resp.StatusCode)
	}
	return true, nil
}
