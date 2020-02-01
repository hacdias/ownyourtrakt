package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"golang.org/x/net/html"
)

type endpoints struct {
	Micropub  string
	IndieAuth string
	Tokens    string
}

func link(doc *html.Node, which string) (string, error) {
	var href string

	var crawler func(node *html.Node)
	crawler = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "link" {
			for _, m := range node.Attr {
				if m.Key == "rel" && m.Val == which {
					for _, m := range node.Attr {
						if m.Key == "href" {
							href = m.Val
							return
						}
					}
				}
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			crawler(child)
		}
	}

	crawler(doc)

	if href == "" {
		return "", errors.New("could not find link tag")
	}

	return href, nil
}

func discoverEndpoints(domain string) (*endpoints, error) {
	resp, err := http.Get(domain)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("code is not 200")
	}

	node, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	micropub, err := link(node, "micropub")
	if err != nil {
		return nil, err
	}

	indieauth, err := link(node, "authorization_endpoint")
	if err != nil {
		return nil, err
	}

	tokens, err := link(node, "token_endpoint")
	if err != nil {
		return nil, err
	}

	return &endpoints{
		Micropub:  micropub,
		IndieAuth: indieauth,
		Tokens:    tokens,
	}, nil
}

type token struct {
	AccessToken string `json:"access_token"`
	Me          string `json:"me"`
	Scope       string `json:"scope"`
}

func getToken(me, code, redirectURI, clientID, codeVerifier, endpoint string) (*token, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("me", me)
	q.Set("grant_type", "authorization_code")
	q.Set("code", code)
	q.Set("redirect_uri", redirectURI)
	q.Set("client_id", clientID)
	q.Set("code_verifier", codeVerifier)

	u.RawQuery = q.Encode()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", u.String(), nil)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res token
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, err
	}

	if res.AccessToken == "" {
		return nil, errors.New("no access token was provided")
	}

	return &res, nil
}
