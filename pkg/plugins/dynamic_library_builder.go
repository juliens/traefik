package plugins

import (
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	_ "embed"
	"text/template"

	"github.com/rs/zerolog/log"
)

//go:embed traefik_go.mod
var goMod []byte

//go:embed traefik_go.sum
var goSum []byte

//go:embed plugin.tmpl
var pluginTmpl []byte

type Plugin struct {
	Hash    string
	Version string
	Import  string
	Name    string
}

func build(plugins []string) (string, error) {
	build, err := os.MkdirTemp("/tmp", "gobuild")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %s", err)
	}

	log.Debug().Msgf("Created temporary directory: %v", build)

	err = os.WriteFile(filepath.Join(build, "go.mod"), goMod, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write go.mod: %s", err)
	}

	log.Debug().Msgf("Created go.mod: %v", goMod)

	err = os.WriteFile(filepath.Join(build, "go.sum"), goSum, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write go.sum: %s", err)
	}

	log.Debug().Msgf("Created go.sum: %v", goSum)

	t := template.New("plugin")
	nT, err := t.Parse(string(pluginTmpl))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %s", err)
	}

	log.Debug().Msgf("Parsed template: %v", nT)

	var pluginsData []Plugin
	for _, plugin := range plugins {
		hasher := fnv.New64()
		hasher.Write([]byte(plugin))
		hash := strconv.FormatUint(hasher.Sum64(), 16)

		parts := strings.Split(plugin, "@")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid plugin name: %s", plugin)
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
		log.Debug().Msgf("Get plugin: %v", plugin)
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
		return "", fmt.Errorf("failed to open plugin.go: %s", err)
	}
	defer f.Close()

	log.Debug().Msgf("Writing plugin.go: %v", pluginsDataNew)

	err = nT.Execute(f, pluginsDataNew)
	if err != nil {
		return "", fmt.Errorf("failed to write plugin.go: %s", err)
	}

	os.Setenv("CGO_ENABLED", "1")

	cmdTidy := exec.Command("go", "mod", "tidy")
	cmdTidy.Dir = build
	cmdTidyOutput, err := cmdTidy.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run go mod tidy: %s %s", cmdTidyOutput, err)
	}

	log.Debug().Msgf("Mod tidy: %v", cmdTidy)

	cmdBuild := exec.Command("go", "build", "-o", "./plugin.so", "-buildmode", "plugin", ".")
	cmdBuild.Dir = build
	cmdBuildOutput, err := cmdBuild.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run go mod build: %s %s", cmdBuildOutput, err)
	}

	log.Debug().Msgf("Mod build: %v", cmdBuildOutput)

	return build + "/plugin.so", nil
}
