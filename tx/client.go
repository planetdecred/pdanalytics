package tx

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	apiTypes "github.com/decred/dcrdata/v7/api/types"
)

type Client struct {
	HttpClient *http.Client
}

func NewClient() (c *Client) {
	return &Client{
		HttpClient: &http.Client{},
	}
}

func (client *Client) GetTransaction(apiBase string, txID string, spends bool) (*apiTypes.Tx, error) {

	tx := &apiTypes.Tx{}

	req, err := http.NewRequest("GET", apiBase+"tx/"+txID+"?spends="+strconv.FormatBool(spends), nil)
	if err != nil {
		return tx, err
	}

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		return tx, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return tx, err
	}

	err = json.Unmarshal(body, &tx)
	if err != nil {
		return tx, err
	}

	return tx, nil
}
