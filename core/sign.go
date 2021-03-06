package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	// 	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

/***
	qcloud cdn openapi
	author:evincai@tencent.com
***/

/**
	*@brief        qcloud cdn openapi signature
	*@param        secretKey    secretKey to log in qcloud
	*@param        params       params of qcloud openapi interface
	*@param        method       http method
	*@param        requesturl   url

	*@return       Signature    signature
	*@return       params       params of qcloud openapi interfac include Signature
**/

var (
	Log = log.New(ioutil.Discard, "", log.LstdFlags)
)

type Client struct {
	httpClient *http.Client
	requesturl string
	verbose    bool
}

func NewClient(requesturl string, verbose bool) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	c := &Client{
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   time.Duration(30) * time.Second,
		},
		requesturl: requesturl,
		verbose:    verbose,
	}
	return c
}

func (c *Client) signature(method string, params map[string]interface{}) map[string]interface{} {
	/*add common params*/
	timestamp := time.Now().Unix()
	rd := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1000)
	params["Timestamp"] = timestamp
	params["Nonce"] = rd
	params["SecretId"] = config.SecretId
	params["SignatureMethod"] = "HmacSHA256"
	/**sort all the params to make signPlainText**/
	sigUrl := method + c.requesturl + "?"
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	isfirst := true
	for _, key := range keys {
		if !isfirst {
			sigUrl = sigUrl + "&"
		}
		isfirst = false
		value := typeSwitcher(params[key])
		if strings.Contains(key, "_") {
			key = strings.Replace(key, "_", ".", -1)
		}
		sigUrl = sigUrl + key + "=" + value
	}
	// 	fmt.Println("signPlainText: ", sigUrl)
	unencode_sign := sign(sigUrl, config.SecretKey)
	params["Signature"] = unencode_sign
	// 	fmt.Println("unencoded signature: ", unencode_sign)
	return params
}

/**
	*@brief        send request to qcloud
	*@param        params       params of qcloud openapi interface include signature
	*@param        method       http method
	*@param        requesturl   url

	*@return       Signature    signature
	*@return       params       params of qcloud openapi interfac include Signature
**/

func fixParams(params map[string]interface{}) {
	for k, v := range params {
		if strings.Contains(k, "_") {
			delete(params, k)
			name := toCamelName(k)
			params[name] = v
		}
	}
}
func toCamelName(name string) string {
	if len(name) == 0 {
		return ""
	}
	part := strings.Split(name, "_")
	var res string
	for _, v := range part {
		v = strings.ToUpper(v[:1]) + v[1:]
		res += v
	}
	res = strings.ToLower(res[:1]) + res[1:]
	return res
}

func (c *Client) SendRequest(method string, params map[string]interface{}) (*http.Response, error) {
	fixParams(params)
	j, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		Log.Println(err.Error())
	}
	Log.Println(string(j))
	p := c.signature(method, params)

	requesturl := "https://" + c.requesturl
	if config.Internal {
		requesturl = "http://" + c.requesturl
	}
	// 	var response string
	if method == "GET" {
		params_str := "?" + ParamsToStr(p)
		requesturl = requesturl + params_str
		// 		response, err := c.httpGet(requesturl)
		return c.httpGet(c.requesturl)
	} else if method == "POST" {
		return c.httpPost(requesturl, p)
	}
	// 		fmt.Println("unsuppported http method")
	return nil, errors.New("unsuppported http method")
	// 	return response
}

func typeSwitcher(t interface{}) string {
	switch v := t.(type) {
	case int:
		return strconv.Itoa(v)
	case string:
		return v
	case int64:
		return strconv.Itoa(int(v))
	default:
		return ""
	}
}

func ParamsToStr(params map[string]interface{}) string {
	isfirst := true
	requesturl := ""
	for k, v := range params {
		if !isfirst {
			requesturl = requesturl + "&"
		}
		isfirst = false
		//		if strings.Contains(k, "_") {
		//			strings.Replace(k, ".", "_", -1)
		//		}
		v := typeSwitcher(v)
		requesturl = requesturl + k + "=" + url.QueryEscape(v)
	}
	return requesturl
}

func sign(signPlainText string, secretKey string) string {
	key := []byte(secretKey)
	hash := hmac.New(sha256.New, key)
	hash.Write([]byte(signPlainText))
	sig := base64.StdEncoding.EncodeToString([]byte(string(hash.Sum(nil))))
	// 	encd_sig := url.QueryEscape(sig)
	return sig
}

func (c *Client) httpGet(url string) (*http.Response, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr, Timeout: time.Duration(30) * time.Second}
	return client.Get(url)
}

func (c *Client) httpPost(requesturl string, params map[string]interface{}) (*http.Response, error) {
	req, err := http.NewRequest("POST", requesturl, strings.NewReader(ParamsToStr(params)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	Log.Println(req.URL.String(), ParamsToStr(params))
	/*
		req, err := http.NewRequest("POST", requesturl, strings.NewReader(form.Encode()))
		fmt.Println(strings.NewReader(form.Encode()))
	*/
	// 	client := &http.Client{Transport: tr, Timeout: time.Duration(3) * time.Second}
	return c.httpClient.Do(req)
}
