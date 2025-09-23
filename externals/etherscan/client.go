package etherscan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	indexer "github.com/feral-file/ff-indexer"
)

const (
	StatusOK    = "1"
	StatusNotOK = "0"

	contentTypeJson           = "application/json"
	contentTypeFormURLEncoded = "application/x-www-form-urlencoded"
)

type Service struct {
	req *Requester
}

type MasterService Service

type Requester struct {
	httpClient http.Client
	baseURL    string
	apiKey     string
}

type Client struct {
	master  *MasterService
	Account *AccountService
}

func NewClient(url string, apiKey string) Client {
	r := &Requester{
		httpClient: http.Client{
			Timeout: time.Minute,
		},
		baseURL: url,
		apiKey:  apiKey,
	}

	c := Client{
		master: &MasterService{
			req: r,
		},
	}
	c.Account = (*AccountService)(c.master)

	return c
}

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Result  interface{} `json:"result"`
}

func (r *Requester) request(
	ctx context.Context,
	method string,
	contentType *string,
	module string,
	query interface{},
	body interface{},
	result interface{}) error {

	// Parse body
	var payload []byte
	if body != nil {
		if nil == contentType {
			ct := contentTypeJson
			contentType = &ct
		}
		switch *contentType {
		case contentTypeJson:
			if b, err := json.Marshal(body); nil != err {
				return err
			} else {
				payload = b
			}
		case contentTypeFormURLEncoded:
			payload = []byte(indexer.BuildQueryParams(body))
		default:
			return errors.New("Unsupported Content-Type: " + *contentType)
		}
	}

	url := fmt.Sprintf("%s?module=%s&apikey=%s",
		r.baseURL,
		module,
		r.apiKey)
	if nil != query {
		url += fmt.Sprintf("&%s", indexer.BuildQueryParams(query))
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		url,
		bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	if nil != contentType {
		req.Header.Add("Content-Type", *contentType)
	}

	// Execute request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Decode the response
	dec := json.NewDecoder(resp.Body)
	var res Response
	if err := dec.Decode(&res); nil != err {
		return err
	}

	// Marshal the result
	jsonVal, err := json.Marshal(res.Result)
	if nil != err {
		return err
	}

	switch res.Status {
	case StatusOK:
		return json.Unmarshal(jsonVal, result)
	case StatusNotOK:
		var errMessage string
		if nil != json.Unmarshal(jsonVal, &errMessage) {
			errMessage = res.Message
		}
		return errors.New(errMessage)
	default:
		return errors.New("unexpected error happened")
	}
}
