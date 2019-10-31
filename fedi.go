package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

type config struct {
	Url      string `json:"url"`
	Token    string `json:"token"`
	Username string `json:"username"`
}

func loadConfig() *config {
	if _, err := os.Stat(".config"); err == nil {
		dat, _ := ioutil.ReadFile(".config")
		c := &config{}
		json.Unmarshal(dat, c)
		return c
	}

	return &config{}
}

func writeConfig(c *config) {
	configMap, _ := json.Marshal(*c)
	_ = ioutil.WriteFile(".config", configMap, 0644)
}

func configure() *config {
	c := loadConfig()

	if c.Url != "" {
		return c
	}

	validate := func(input string) error {
		if !strings.HasPrefix(input, "https://") {
			return errors.New("Must begin with https://")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Instance URL",
		Validate: validate,
	}

	result, err := prompt.Run()
	if err != nil {
		fmt.Printf("%v", err)
	}

	c.Url = result

	return c
}

func createClient(c *config) map[string]interface{} {
	client := &http.Client{}
	u, _ := url.Parse(c.Url)
	u.Path = path.Join(u.Path, "/api/v1/apps")
	data := url.Values{}
	data.Set("client_name", "fedigo")
	data.Set("redirect_uris", "urn:ietf:wg:oauth:2.0:oob")
	data.Set("scopes", "read write follow")

	req, _ := http.NewRequest("POST", u.String(), strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, _ := client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)

	var v map[string]interface{}

	json.Unmarshal(body, &v)
	return v
}

func authenticate(c *config) {
	if c.Token != "" {
		return
	}

	client := &http.Client{}

	authClient := createClient(c)
	clientId := authClient["client_id"]
	clientSecret := authClient["client_secret"]

	validate := func(input string) error {
		if len(input) == 0 {
			return errors.New("Cannot be empty")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Username",
		Validate: validate,
	}

	result, _ := prompt.Run()
	c.Username = result

	prompt = promptui.Prompt{
		Label:    "Password",
		Validate: validate,
		Mask:     '*',
	}

	result, _ = prompt.Run()
	password := result

	u, _ := url.Parse(c.Url)
	u.Path = path.Join(u.Path, "/oauth/token")
	data := url.Values{}
	data.Set("client_id", clientId.(string))
	data.Set("client_secret", clientSecret.(string))
	data.Set("username", c.Username)
	data.Set("password", password)
	data.Set("grant_type", "password")
	data.Set("scope", "read write follow")

	req, _ := http.NewRequest("POST", u.String(), strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, _ := client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	var v map[string]interface{}
	json.Unmarshal(body, &v)
	c.Token = fmt.Sprintf("Bearer %s", v["access_token"])
}

func postLoop(c *config) {
	client := &http.Client{}

	prompt := promptui.Prompt{
		Label: "Post",
	}
	status, _ := prompt.Run()
	visibilityPrompt := promptui.Select{
		Label: "Visibility",
		Items: []string{"public", "unlisted", "private", "direct"},
	}
	_, visibility, _ := visibilityPrompt.Run()

	u, _ := url.Parse(c.Url)
	u.Path = path.Join(u.Path, "/api/v1/statuses")
	data := url.Values{}
	data.Set("status", status)
	data.Set("visibility", visibility)

	req, _ := http.NewRequest("POST", u.String(), strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", c.Token)

	resp, _ := client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	var v map[string]interface{}
	json.Unmarshal(body, &v)
	fmt.Println(v["url"])
}

func main() {
	conf := configure()
	authenticate(conf)
	writeConfig(conf)
	for {
		postLoop(conf)
	}
}
