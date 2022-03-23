package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

const rootURL = "mypsw"

func handle(w http.ResponseWriter, req *http.Request) {
	upath := req.URL.Path
	if strings.HasPrefix(upath, "/static/") {
		http.ServeFile(w, req, path.Clean(rootURL+upath))
	}
	value, ok := htms[filepath.FromSlash(rootURL+upath+".page.htm")]
	if ok {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(value))
	} else {
		if upath == "/" {
			upath = ""
		}
		value, ok := htms[filepath.FromSlash(rootURL+upath+"/index.page.htm")]
		if ok {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(value))
		} else {
			w.WriteHeader(404)
		}
	}
}

var htms = map[string]string{}

func init() {
	root := "./" + rootURL + "/"
	var layparts []string
	var laypartsJSON []string

	err := filepath.Walk(root, func(upath string, info os.FileInfo, err error) error {
		if strings.HasSuffix(upath, ".layout.htm") || strings.HasSuffix(upath, ".partial.htm") {
			layparts = append(layparts, upath)
			if _, err := os.Stat(upath[0:len(upath)-4] + ".json"); !os.IsNotExist(err) {
				laypartsJSON = append(laypartsJSON, upath[0:len(upath)-4]+".json")
			}
		}
		return nil
	})

	if err != nil {
		panic(err)
	}

	err = filepath.Walk(root, func(upath string, info os.FileInfo, err error) error {
		spew.Dump(upath)
		if strings.HasSuffix(upath, ".page.htm") {
			files := append([]string{upath}, layparts...)
			spew.Dump(files)
			t, err := template.ParseFiles(files...)
			if err != nil {
				panic(err)
			}
			var tpl bytes.Buffer
			var params map[string]interface{}
			for _, lpj := range laypartsJSON {
				jsonFile, err := os.Open(lpj)
				if err != nil {
					panic(err)
				}
				defer jsonFile.Close()
				byteValue, _ := ioutil.ReadAll(jsonFile)
				json.Unmarshal(byteValue, &params)
			}
			if _, err := os.Stat(upath[0:len(upath)-4] + ".json"); !os.IsNotExist(err) {
				jsonFile, err := os.Open(upath[0:len(upath)-4] + ".json")
				if err != nil {
					panic(err)
				}
				defer jsonFile.Close()
				byteValue, _ := ioutil.ReadAll(jsonFile)
				json.Unmarshal(byteValue, &params)
			}
			err = t.Execute(&tpl, params)
			if err != nil {
				panic(err)
			}
			htms[strings.ToLower(upath)] = tpl.String()
		}
		return nil
	})

	if err != nil {
		panic(err)
	}
}

func main() {
	srv := &http.Server{Addr: ":7777", Handler: http.HandlerFunc(handle)}
	log.Printf("Serving on https://127.0.0.1:7777")
	log.Fatal(srv.ListenAndServe())
}
