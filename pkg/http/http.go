package http

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	log "k8s.io/klog"

	"ezone.ksyun.com/ezone/kce/vpc-route-controller/pkg/util"
)

type Client struct {
	modifiers []Modifier
	client    *http.Client
}

func NewClient(c *http.Client, modifiers ...Modifier) *Client {
	client := &Client{
		client: c,
	}
	if client.client == nil {
		client.client = &http.Client{}
	}
	if len(modifiers) > 0 {
		client.modifiers = modifiers
	}
	return client
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.modifiers != nil {
		for _, modifier := range c.modifiers {
			if modifier == nil {
				continue
			}
			if err := modifier.Modify(req); err != nil {
				log.Errorf("modifier modify request error :%v", err)
				return nil, err
			}
		}
	}
	return c.client.Do(req)
}

func (c *Client) PostJson(url string, v ...interface{}) ([]byte, error) {
	var reader io.Reader
	if len(v) > 0 {
		data, err := json.Marshal(v[0])
		if err != nil {
			log.Errorf("json marshal error :%v", err)
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	return c.post(url, "application/json", reader)
}

func (c *Client) PostForm(url string, paramMap map[string]string) ([]byte, error) {
	param := util.Encode(paramMap)
	log.Infof("post form param :%v", param)
	var reader io.Reader
	if len(param) > 0 {
		reader = strings.NewReader(param)
	}
	return c.post(url, "application/x-www-form-urlencoded", reader)
}

func (c *Client) post(url string, contentType string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		log.Errorf("post json, create request error :%v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	byteData, err := c.DoRequest(req)
	return byteData, err
}

func (c *Client) DoRequest(req *http.Request) ([]byte, error) {
	log.Infof("http request info :%v", *req)
	resp, err := c.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Errorf("do request error :%v", err)
		return nil, err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("read body error :%v", data)
		return nil, err
	}
	log.Infof("http response body :%v", string(data))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, &Error{
			Code:    resp.StatusCode,
			Message: string(data),
		}
	}
	return data, nil
}
