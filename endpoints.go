package main

import (
	"context"
	"errors"
	"net/http"
	"time"

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
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", domain, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
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
