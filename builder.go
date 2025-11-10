package main

import (
	"bufio"
	_ "embed"
	"flag"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

//go:embed go.mod
var goMod []byte

//go:embed go.sum
var goSum []byte

type Plugin struct {
	Hash    string
	Version string
	Import  string
	Name    string
}

func main() {
	var listen string
	flag.StringVar(&listen, "listen", ":8080", "listen address")
	flag.Parse()

	err := http.ListenAndServe(listen, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var plugins []string
		if req.Method == "GET" {
			plugin := req.URL.Query().Get("plugin")
			if plugin == "" {
				http.Error(rw, "plugin required", http.StatusBadRequest)
				return
			}

			plugins = append(plugins, plugin)
		}

		if req.Method == "POST" {
			defer req.Body.Close()

			scanner := bufio.NewScanner(req.Body)
			for scanner.Scan() {
				line := scanner.Text()
				fmt.Println("LINE", line)

				plugins = append(plugins, line)
			}

			if err := scanner.Err(); err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		src, err := build(plugins)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			log.Println(err)
			return
		}
		rw.Write(src)
	}))

	if err != nil {
		log.Fatal(err)
	}
}

func build(plugins []string) ([]byte, error) {
	build, err := os.MkdirTemp("/tmp", "gobuild")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %s", err)
	}

	err = os.WriteFile(filepath.Join(build, "go.mod"), goMod, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write go.mod: %s", err)
	}

	err = os.WriteFile(filepath.Join(build, "go.sum"), goSum, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write go.sum: %s", err)
	}

	b, err := os.ReadFile("./plugin.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin.tmpl: %s", err)
	}

	t := template.New("plugin")
	nT, err := t.Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %s", err)
	}

	var pluginsData []Plugin
	for _, plugin := range plugins {
		hasher := fnv.New64()
		hasher.Write([]byte(plugin))
		hash := strconv.FormatUint(hasher.Sum64(), 16)

		parts := strings.Split(plugin, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid plugin name: %s", plugin)
		}

		plugin = parts[0]
		version := parts[1]

		pluginName := plugin
		parts = strings.Split(plugin, "/")
		if len(parts) > 3 {
			pluginName = strings.Join(parts[0:2], "/")
		}

		pluginsData = append(pluginsData, Plugin{
			Hash:    "p" + hash,
			Name:    pluginName,
			Import:  plugin,
			Version: version,
		})
	}

	var pluginsDataNew []Plugin
	for _, plugin := range pluginsData {
		cmdTidy := exec.Command("go", "get", plugin.Name+"@"+plugin.Version)
		cmdTidy.Dir = build
		cmdTidyOutput, err := cmdTidy.CombinedOutput()
		if err != nil {
			fmt.Println(fmt.Errorf("failed to run go get: %s %s", cmdTidyOutput, err))
			continue
		}
		pluginsDataNew = append(pluginsDataNew, plugin)
	}

	f, err := os.OpenFile(build+"/plugin.go", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin.go: %s", err)
	}
	defer f.Close()

	err = nT.Execute(f, pluginsDataNew)
	if err != nil {
		return nil, fmt.Errorf("failed to write plugin.go: %s", err)
	}

	os.Setenv("CGO_ENABLED", "1")

	cmdTidy := exec.Command("go", "mod", "tidy")
	cmdTidy.Dir = build
	cmdTidyOutput, err := cmdTidy.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run go mod tidy: %s %s", cmdTidyOutput, err)
	}

	cmdBuild := exec.Command("go", "build", "-o", "./plugin.so", "-buildmode", "plugin", ".")
	cmdBuild.Dir = build
	cmdBuildOutput, err := cmdBuild.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run go mod build: %s %s", cmdBuildOutput, err)
	}

	return os.ReadFile(build + "/plugin.so")
}
