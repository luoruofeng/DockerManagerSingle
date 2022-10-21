package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func setReqHeader(outter *http.Request, target *http.Request) {
	for k, v := range outter.Header {
		if len(v) <= 1 {
			target.Header.Add(k, v[0])
		} else {
			target.Header.Add(k, strings.Join(v, ","))
		}
	}
}

func setRespHeader(resp *http.Response, w *http.ResponseWriter) {
	for k, v := range resp.Header {
		if len(v) <= 1 {
			(*w).Header().Add(k, v[0])
		} else {
			(*w).Header().Add(k, strings.Join(v, ","))
		}
	}
}

func setReqCookie(target *http.Request, outter *http.Request) {
	for _, c := range outter.Cookies() {
		target.AddCookie(c)
	}
}

func setRespCookie(target *http.Response, outter *http.ResponseWriter) {
	for _, c := range target.Cookies() {

		http.SetCookie(*outter, c)
	}

}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	url := r.URL
	url.Host = "172.17.0.3:80"
	if url.Scheme == "" {
		url.Scheme = "http"
	}
	fmt.Println(url.String())
	nr, err := http.NewRequest(r.Method, url.String(), r.Body)
	if err != nil {
		fmt.Println(err)
	}

	setReqHeader(r, nr)
	setReqCookie(nr, r)
	resp, err := http.DefaultClient.Do(nr)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	setRespHeader(resp, &w)
	setRespCookie(resp, &w)
	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}

func Start(ctx context.Context, cancelFunc func(), port int) {
	r := mux.NewRouter()
	r.PathPrefix("/").HandlerFunc(proxyHandler)
	fmt.Println(http.ListenAndServe(":"+strconv.Itoa(port), r))
}
