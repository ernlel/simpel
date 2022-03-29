package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"html/template"
	"io/fs"
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

var rootURL string
var staticsFolder string
var restartOnChange bool
var port string

func handle(w http.ResponseWriter, req *http.Request) {
	upath := req.URL.Path
	if upath != "/" {
		upath = upath + "/"
	}
	if strings.HasPrefix(upath, "/"+staticsFolder+"/") {
		http.ServeFile(w, req, path.Clean(rootURL+upath))
		return
	}

	value, ok := htms[filepath.FromSlash(rootURL+upath+".page.htm")]
	if ok {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(value))
		return
	}

	value, ok = htms[filepath.FromSlash(rootURL+upath+"index.page.htm")]
	if ok {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(value))
		return
	}

	w.WriteHeader(404)

}

var htms = map[string]string{}

func init() {
	flag.BoolVar(&restartOnChange, "watch", false, "Watch templates and restart if change detected. Default 'false'")
	flag.StringVar(&rootURL, "template", "/home/ernest/Documents/Projects/portfolio/web/", "Path to templates folder. Default './example'")
	flag.StringVar(&staticsFolder, "static", "assets", "name of static folder. Default 'assets'")
	flag.StringVar(&port, "port", "7777", "Port. Default '7777'")
	flag.Parse()
	rootURL = path.Clean(rootURL)

	var layparts []string
	var laypartsJSON []string

	// walk and firstly add layouts and partials
	err := filepath.WalkDir(rootURL, func(upath string, _ fs.DirEntry, err error) error {
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

	// walk again and add pages (if walk in one go, so some pages will be missing layouts and partials)
	err = filepath.WalkDir(rootURL, func(upath string, _ fs.DirEntry, err error) error {
		if strings.HasSuffix(upath, ".page.htm") {
			files := append([]string{upath}, layparts...)
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
			htms[upath] = tpl.String()
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

	srv := &http.Server{Addr: ":" + port, Handler: http.HandlerFunc(handle)}
	log.Printf("Serving on http://127.0.0.1:" + port)
	log.Fatal(srv.ListenAndServe())
}
