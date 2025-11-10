package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/traefik/traefik/v3/pkg/plugins"
)

func main() {
	var listen string
	flag.StringVar(&listen, "listen", ":80", "listen address")
	flag.Parse()

	err := http.ListenAndServe(listen, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var pluginsList []string
		if req.Method == "GET" {
			plugin := req.URL.Query().Get("plugin")
			if plugin == "" {
				http.Error(rw, "plugin required", http.StatusBadRequest)
				return
			}

			pluginsList = append(pluginsList, plugin)
		}

		if req.Method == "POST" {
			defer req.Body.Close()

			scanner := bufio.NewScanner(req.Body)
			for scanner.Scan() {
				line := scanner.Text()

				pluginsList = append(pluginsList, line)
			}

			if err := scanner.Err(); err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if strings.Contains(req.URL.Path, "/report") {
			report := plugins.ReportErrors(pluginsList)

			rw.Write([]byte(report))
			return
		}

		src, err := plugins.BuildSO(pluginsList)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}

		f, err := os.Open(src)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
		}

		defer f.Close()
		io.Copy(rw, f)
	}))

	if err != nil {
		log.Fatal(err)
	}
}
