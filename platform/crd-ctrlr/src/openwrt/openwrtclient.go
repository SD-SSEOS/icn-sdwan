// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2021 Intel Corporation

package openwrt

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"strings"
	"sync"
)

type IOpenWrtObject interface {
	GetName() string
	SetFullName(string)
}

type OpenwrtError struct {
	Code    int
	Message string
}

func (e *OpenwrtError) Error() string {
	return fmt.Sprintf("Error Code: %d, Error Message: %s", e.Code, e.Message)
}

type OpenwrtClientInfo struct {
	Ip       string
	User     string
	Password string
	RootCA   []byte
}

type openwrtClient struct {
	OpenwrtClientInfo
	caCertPool *x509.CertPool
	token      string
}

type safeOpenwrtClient struct {
	clients map[string]*openwrtClient
	mux     sync.Mutex
}

var gclients = safeOpenwrtClient{clients: make(map[string]*openwrtClient)}

func CloseClient(o *openwrtClient) {
	err := o.logout()
	if err != nil {
		log.Println(err)
	}
	runtime.SetFinalizer(o, nil)
}

func GetOpenwrtClient(clientInfo OpenwrtClientInfo) *openwrtClient {
	return gclients.GetClient(clientInfo.Ip, clientInfo.User, clientInfo.Password, clientInfo.RootCA)
}

// SafeOpenwrtClients
func (s *safeOpenwrtClient) GetClient(ip string, user string, password string, rootCA []byte) *openwrtClient {
	s.mux.Lock()
	defer s.mux.Unlock()
	key := ip + "-" + user + "-" + password
	if s.clients[key] == nil {
		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(rootCA)
		if !ok {
			log.Println("Error to create rootCA")
		}

		s.clients[key] = &openwrtClient{
			OpenwrtClientInfo: OpenwrtClientInfo{
				Ip:       ip,
				User:     user,
				Password: password,
				RootCA:   rootCA,
			},
			caCertPool: caCertPool,
			token:      "",
		}
	}

	return s.clients[key]
}

// openwrt base URL
func (o *openwrtClient) getBaseURL() string {
	return "https://" + o.Ip + "/cgi-bin/luci/"
}

// login to openwrt http server
func (o *openwrtClient) login() error {
	if o.Password == "" {
		return &OpenwrtError{Code: 403, Message: "Unauthorized"}
	}
	client := &http.Client{
		// block redirect
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: o.caCertPool,
			},
		},
	}

	// login
	login_info := "luci_username=" + o.User + "&luci_password=" + o.Password
	var req_body = []byte(login_info)
	req, _ := http.NewRequest("POST", o.getBaseURL(), bytes.NewBuffer(req_body))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	} else if resp.StatusCode != 302 {
		// fail to auth
		return &OpenwrtError{Code: resp.StatusCode, Message: "Unauthorized"}
	} else {
		// get token
		res_cookie := resp.Header["Set-Cookie"][0]
		res_cookies := strings.Split(res_cookie, ";")
		for _, cookie := range res_cookies {
			cookie := strings.TrimSpace(cookie)
			index := strings.Index(cookie, "=")
			var key = cookie
			var value = ""
			if index != -1 {
				key = cookie[:index]
				value = cookie[index+1:]
			}

			if key == "sysauth" {
				o.token = value
				break
			}
		}
	}

	return nil
}

// logout to openwrt http server
func (o *openwrtClient) logout() error {
	if o.token != "" {
		_, err := o.Get("admin/logout")
		o.token = ""
		return err
	}

	return nil
}

// call openwrt restful API
func (o *openwrtClient) call(method string, url string, request string) (string, error) {
	for i := 0; i < 2; i++ {
		if o.token == "" {
			err := o.login()
			if err != nil {
				return "", err
			}
		}

		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: o.caCertPool,
				},
			},
		}

		req_body := bytes.NewBuffer([]byte(request))
		req, _ := http.NewRequest(method, o.getBaseURL()+url, req_body)
		req.Header.Add("Cookie", "sysauth="+o.token)
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode >= 400 {
			if resp.StatusCode == 403 {
				// token expired, retry
				o.token = ""
				continue
			} else {
				// error request
				return "", &OpenwrtError{Code: resp.StatusCode, Message: string(body)}
			}
		}

		return string(body), nil
	}

	return "", nil
}

// call openwrt Get restful API
func (o *openwrtClient) Get(url string) (string, error) {
	return o.call("GET", url, "")
}

// call openwrt restful API
func (o *openwrtClient) Post(url string, request string) (string, error) {
	return o.call("POST", url, request)
}

// call openwrt restful API
func (o *openwrtClient) Put(url string, request string) (string, error) {
	return o.call("PUT", url, request)
}

// call openwrt restful API
func (o *openwrtClient) Delete(url string) (string, error) {
	return o.call("DELETE", url, "")
}
