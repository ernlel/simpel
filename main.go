package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

const rootURL = "example"
const staticsFolder = "assets"

func handle(w http.ResponseWriter, req *http.Request) {
	upath := req.URL.Path
	if strings.HasPrefix(upath, "/"+staticsFolder+"/") {
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
		// spew.Dump(upath)
		if strings.HasSuffix(upath, ".page.htm") {
			files := append([]string{upath}, layparts...)
			// spew.Dump(files)
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

func RestartSelf() error {
	self, err := os.Executable()
	if err != nil {
		return err
	}
	args := os.Args
	env := os.Environ()
	// For Windows
	if runtime.GOOS == "windows" {
		cmd := exec.Command(self, args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Env = env
		err := cmd.Run()
		if err == nil {
			os.Exit(0)
		}
		return err
	}
	return syscall.Exec(self, args, env)
}

func main() {
	var restartOnChange bool
	flag.BoolVar(&restartOnChange, "w", false, "Watch templates and restart if change detected. Default is false")
	flag.Parse()

	if restartOnChange {
		nestedWatchItems, err := NestedWatch("./" + rootURL)
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			for {
				nestedWatchItem := <-nestedWatchItems
				log.Println(nestedWatchItem)
				RestartSelf()
			}
		}()
	}

	srv := &http.Server{Addr: ":7777", Handler: http.HandlerFunc(handle)}
	log.Printf("Serving on http://127.0.0.1:7777")
	log.Fatal(srv.ListenAndServe())
}
