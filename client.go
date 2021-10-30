package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/v32/github"
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

type ghpClient struct {
	oauthToken string
	deviceCode string
	apiClient  *github.Client
	context    *context.Context
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

func (c *ghpClient) prepareDeviceForOauth() (string, string, error) {
	device, err := oauthCreateDeviceRequest()
	if err != nil {
		return "", "", fmt.Errorf("error creating device request: %v", err)
	}
	c.deviceCode = device.DeviceCode
	return device.UserCode, device.VerificationURI, nil
}

func (c *ghpClient) performOauth() error {
	body := strings.NewReader(fmt.Sprintf("client_id=0412cc5fb93b10a59e50&device_code=%s&grant_type=urn:ietf:params:oauth:grant-type:device_code", c.deviceCode))

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var responseAuth oauthAuthCodeResponse
	err = json.Unmarshal(responseBody, &responseAuth)
	if err != nil {
		return err
	}
	c.oauthToken = responseAuth.AccessToken
	return nil
}

func createClient(authToken string) *ghpClient {
	c := new(ghpClient)
	c.oauthToken = authToken
	ctx := context.Background()
	c.context = &ctx
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: c.oauthToken},
	)
	tc := oauth2.NewClient(*c.context, ts)
	c.apiClient = github.NewClient(tc)
	return c
}

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

func (c *ghpClient) getAllColumnCards(columnId int64) ([]*github.ProjectCard, error) {
	cards, res, err := c.apiClient.Projects.ListProjectCards(*c.context, columnId, nil)
	if err != nil {
		return nil, fmt.Errorf("error Getting cards for %v: %v", columnId, err)
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("error Getting cards for %v: http: %v", columnId, res.Status)
	}
	return cards, nil
}

func (c *ghpClient) listColumns(projectID int64) ([]*github.ProjectColumn, error) {
	cols, res, err := c.apiClient.Projects.ListProjectColumns(*c.context, projectID, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting columns for %v: %v", projectID, err)
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("error getting columns for %v: http: %v", projectID, res.Status)
	}

	return cols, nil
}

func (c *ghpClient) getAPIObject(url string, v interface{}) error {
	req, _ := c.apiClient.NewRequest("GET", url, nil)
	res, err := c.apiClient.Do(*c.context, req, v)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("error getting %v: %v", url, res.Status)
	}
	return nil
}
