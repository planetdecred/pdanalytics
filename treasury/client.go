package treasury

import (
	"net/http"
)

type Client struct {
	HttpClient *http.Client
}

func NewClient() (c *Client) {
	return &Client{
		HttpClient: &http.Client{},
	}
}
