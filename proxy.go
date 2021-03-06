package main

import (
	"fmt"
	"flag"
	"log"
	"net/http"
	"net/url"
	"strings"
	"io"
	"io/ioutil"
	"encoding/json"
	"crypto/tls"	
)

type commitData struct {
	Before     string
	After      string
	Ref        string
	UserName   string
	Repository struct {
		Url string
	}
}

var (
	listen = flag.String("listen", "localhost:9080", "listen on address")
	logp   = flag.Bool("log", false, "enable logging")
)

func main() {
	flag.Parse()
	proxyHandler := http.HandlerFunc(proxyHandlerFunc)
	log.Fatal(http.ListenAndServe(*listen, proxyHandler))
}

func readerToString(r io.Reader) string {
	if b, err := ioutil.ReadAll(r); err == nil {
		return string(b)
	} 
	return ""
}

func setGitData(form url.Values, g commitData) {
	form.Set("before", g.Before)
	form.Set("after", g.After)
	form.Set("ref", g.Ref)
	
	refToWork := g.Ref
	s := strings.Split(refToWork, "/");
	log.Printf("Tag is : %v\n", s[2])
	
	form.Set("tag_name", s[2])
	form.Set("git_url", g.Repository.Url)
}

func proxyToEndpoint(url string, form url.Values, w http.ResponseWriter) error {

	tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    client := &http.Client{Transport: tr}
	resp, err := client.PostForm(url, form)
	log.Printf("Posting to: %v\n", url)
	log.Printf("Posting to: %v\n", form)

	if err != nil {
		log.Print(err)
		fmt.Fprintf(w, "ERROR")
	} else {
		defer resp.Body.Close()
		resp.Write(w)
	}
	return err
}

func infoPage(notice string) string {
	return fmt.Sprintf(
		"<html><body><h1>githookproxy</h1>"+
			"<p>Proxy takes JSON body in the format of: </p>"+
			"<p><a href='http://grab.by/qrKw'/>Gitlab Webhook</a></p>"+
			"<p>It will converts it to parameters and will post to url specified by 'url' param.</p>"+
			"<p>Parameters will include:"+
			"<ul><li>payload:JSON body</li><li>URL: url of git repo</li>"+
			"<li>START: Start commit hash</li><li>END: End commit hash</li>"+
			"<li>REFNAME: Ref name</li></ul></p>"+
			"<p>To use, add this to your Gitlab webook: http://[proxy_listen_url]?url=[target_url]</p>"+
			"<p><strong>Notice: %v</strong></p>"+
			"<p>Code: <a href='https://github.com/akira/githookproxy'>Github</a></html></body>",
		notice)
}

func proxyHandlerFunc(w http.ResponseWriter, r *http.Request) {
	if *logp {
		log.Println(r.URL)
	}

	body := readerToString(r.Body)
	decoder := json.NewDecoder(strings.NewReader(body))
	var gitData commitData
	err := decoder.Decode(&gitData)

	if err != nil {
		log.Print(err)
		fmt.Fprintf(w, infoPage("JSON body not found or invalid!"))
	} else if r.FormValue("url") == "" {
		log.Print("URL not found!")
		fmt.Fprintf(w, infoPage("URL not found!"))
	} else {
		form := make(url.Values)
		setGitData(form, gitData)
		form.Set("PAYLOAD", body)

		postUrl := r.FormValue("url")
		proxyToEndpoint(postUrl, form, w)
	}
}
