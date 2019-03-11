package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

/**
 * Accepts a single HTTP request, forwards to dst, returns response via w
 */
func handler(destSockPath string, requestPreProcessor func(*http.Request) bool) func(w http.ResponseWriter, r *http.Request) {
	myf := func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logger.Errorf("failed to read body")
			return
		}
		logger.Infof("%s %s %s\n", r.Method, r.URL.String(), r.Proto)

		httpc := http.Client{
			// this http.Client.Transport override for unix domain socket adapted from:
			// https://gist.github.com/teknoraver/5ffacb8757330715bcbcc90e6d46ac74
			Transport: &http.Transport{
				DialContext: func(c context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", destSockPath)
				},
			},
		}

		// make a new request to send to actual socket
		var dstReq *http.Request
		var dstContentLength int
		logger.Infof("request url: " + r.URL.String())
		if method == "GET" {
			dstReq, err = http.NewRequest(
				method,
				"http://unix"+r.URL.String(), // URL is the path in server
				bytes.NewReader(body))
			dstContentLength = len(body)
		} else if method == "POST" || method == "PUT" {
			r.ParseForm()
			dstForm := make(url.Values)
			formValues := r.PostForm
			for k, v := range formValues {
				val := strings.Join(v, " ")
				dstForm.Set(k, strings.TrimSpace(val))
			}
			formStr := dstForm.Encode()
			logger.Debugf("form: " + formStr)
			dstReq, err = http.NewRequest(
				method,
				"http://unix"+r.URL.String(), // URL is the path in server
				strings.NewReader(formStr))
			dstContentLength = len(formStr)
		}
		//dstReq.Header = make(http.Header)
		for h, val := range r.Header {
			h = strings.ToLower(h)
			for _, v := range val {
				logger.Debugf("header: %v: %v\n", h, v)
				dstReq.Header.Add(h, strings.TrimSpace(v))
			}
		}
		dstCLenStr := strconv.FormatInt(int64(dstContentLength), 10)
		logger.Debugf("setting content-length to %s", dstCLenStr)
		dstReq.Header.Set("content-length", dstCLenStr)

		// preprocess with provided function, to modify values before proxying
		changed := requestPreProcessor(dstReq)
		if changed {
			logger.Infof("requestPreProcessor changed the request")
			logger.Infof("%v\n", dstReq.URL)
		}

		// send proxy request to destination, and get response
		res, err := httpc.Do(dstReq)
		if err != nil {
			logger.Errorf("failed to do proxy req")
		}

		// process response
		resBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			logger.Errorf("failed to read response body")
			return
		}

		// copy headers into new response
		for h_key, h_val := range res.Header {
			h_key = strings.ToLower(h_key)
			if h_key == "content-length" {
				contentLength, _ := strconv.ParseInt(h_val[0], 10, 32)
				if int(contentLength) != len(resBody) {
					logger.Errorf(
						"content-length header was %d but length of body was %d",
						int(contentLength),
						len(resBody))
				}
				w.Header().Set(h_key, strings.Join(h_val, ","))
			} else {
				w.Header()[h_key] = h_val //strings.Join(value, ",")
			}
		}
		// content-length header should already be set, but set from the field as well
		//w.Header().Set("content-length", strconv.FormatInt(res.ContentLength, 10))
		resStatusCode := res.StatusCode
		logger.Debugf("RESPONSE:")
		logger.Debugf(string(resBody))
		w.WriteHeader(int(resStatusCode))

		w.Write([]byte(resBody))
	}
	return myf
}

func unix_domain_socket_proxy(srcPath string, dstPath string, reqPreProcessor func(req *http.Request) bool) {
	defaultRequestPreProcessor := func(req *http.Request) bool {
		// modify nothing
		return false
	}
	if reqPreProcessor == nil {
		reqPreProcessor = defaultRequestPreProcessor
	}
	l, err := net.Listen("unix", srcPath)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	http.HandleFunc("/", handler(dstPath, reqPreProcessor))
	logger.Infof("starting server")
	if err := http.Serve(l, nil); err != nil {
		log.Fatal(err)
	}
}
