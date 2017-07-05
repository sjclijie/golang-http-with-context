package main

import (
	"net/http"
	"time"
	"context"
	"fmt"
	"net"
	"encoding/json"
)

type key int

const userIPKey key = 0

func getIPFromRequest(r *http.Request) (net.IP, error) {

	host, _, err := net.SplitHostPort(r.RemoteAddr)

	if err != nil {
		return nil, err
	}

	userIP := net.ParseIP(host)

	if userIP == nil {
		return nil, fmt.Errorf("remote addr error , %s", r.RemoteAddr)
	}

	return userIP, nil
}

func NewContextWithIP(ctx context.Context, userIP net.IP) context.Context {
	return context.WithValue(ctx, userIPKey, userIP)
}

type result struct {
	Timestamp string `json:"timestamp"`
	Expire    string `json:"expire"`
	Scheme    string `json:"scheme"`
	ImageUrl  string `json:"image_url"`
}

const API string = "http://api.suiyueyule.com/1.0.2/config/splash"

func search(ctx context.Context, query string) (result, error) {

	var result result

	req, err := http.NewRequest(http.MethodGet, API, nil)

	if err != nil {
		return result, err
	}

	q := req.URL.Query();
	if userIP, err := ctx.Value(userIPKey).(net.IP); err {
		q.Set("userip", userIP.String())
	}

	q.Set("q", query)

	req.URL.RawQuery = q.Encode()

	err = doRequest(ctx, req, func(response *http.Response, err error) error {

		if err != nil {
			return nil
		}

		defer response.Body.Close()

		var data struct {
			Status int `json:"status"`
			Msg    string `json:"msg"`
			Data struct {
				Timestamp string `json:"timestamp"`
				Expire    string `json:"expire"`
				Scheme    string `json:"scheme"`
				ImageUrl  string `json:"image_url"`
			}
		}

		if err := json.NewDecoder(response.Body).Decode(&data); err != nil {
			return err
		}

		fmt.Println(data.Data)

		return nil
	})

	return result, err
}

func doRequest(ctx context.Context, r *http.Request, f func(*http.Response, error) error) error {

	tr := &http.Transport{}
	client := &http.Client{Transport: tr }

	c := make(chan error, 1)

	go func() {
		c <- f(client.Do(r))
	}()

	select {
	case <-ctx.Done():
		fmt.Println("优先ctx.done")
		tr.CancelRequest(r)
		<-c
		return ctx.Err()
	case err := <-c:
		fmt.Println("优先result")
		return err
	}
}

func handleSearch(w http.ResponseWriter, r *http.Request) {

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond*50)

	/*
	timeout, err := time.ParseDuration(r.FormValue("timeout"))

	if err != nil {
		fmt.Println(r.FormValue("timeout"))
		ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond * 50 )
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	*/

	defer cancel()

	query := r.FormValue("q")

	if query == "" {
		http.Error(w, "no query", http.StatusNotAcceptable)
		return
	}

	userIP, err := getIPFromRequest(r)

	ctx = NewContextWithIP(ctx, userIP)

	start := time.Now()

	result, err := search(ctx, query)

	fmt.Println(result, err, start);

	fmt.Fprint(w, err)
}

func main() {
	http.HandleFunc("/search", handleSearch)
	http.ListenAndServe(":9988", nil)
}
