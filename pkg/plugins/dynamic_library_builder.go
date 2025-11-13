package plugins

import (
	_ "embed"
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/rs/zerolog/log"
)

//go:embed traefik_go.mod
var goMod []byte

//go:embed traefik_go.sum
var goSum []byte

//go:embed plugin.tmpl
var pluginTmpl []byte

var t *template.Template

type Plugin struct {
	Hash    string
	Version string
	Import  string
	Name    string
}

func init() {
	var err error
	t, err = template.New("plugin").Parse(string(pluginTmpl))
	if err != nil {
		panic(err)
	}
}

func initDir() string {
	build, err := os.MkdirTemp("/tmp", "gobuild")
	if err != nil {
		return ""
	}

	log.Debug().Msgf("Created temporary directory: %v", build)

	err = os.WriteFile(filepath.Join(build, "go.mod"), goMod, 0644)
	if err != nil {
		return ""
	}

	err = os.WriteFile(filepath.Join(build, "go.sum"), goSum, 0644)
	if err != nil {
		return ""
	}

	log.Debug().Msgf("Created go.sum and go.mod")

	return build
}

func buildPlugin(build string, pluginsData []Plugin) error {
	err := createGoCode(build, pluginsData)
	if err != nil {
		return err
	}

	os.Setenv("CGO_ENABLED", "1")

	cmdTidy := exec.Command("go", "mod", "tidy")
	cmdTidy.Dir = build
	cmdTidyOutput, err := cmdTidy.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run go mod tidy: %s %s", cmdTidyOutput, err)
	}

	log.Debug().Msgf("Mod tidy: %v", cmdTidy)

	cmdBuild := exec.Command("go", "build", "-o", "./plugin.so", "-buildmode", "plugin", ".")
	cmdBuild.Dir = build
	cmdBuildOutput, err := cmdBuild.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run go build: %s %s", cmdBuildOutput, err)
	}

	log.Debug().Msgf("Mod build: %v", cmdBuildOutput)

	return nil
}

func getPluginsData(plugins []string) ([]Plugin, error) {
	var pluginsData []Plugin
	for _, plugin := range plugins {
		hasher := fnv.New64()
		hasher.Write([]byte(plugin))
		hash := strconv.FormatUint(hasher.Sum64(), 16)

		parts := strings.Split(plugin, "=>")
		importPath := ""
		if len(parts) == 2 {
			importPath = parts[1]
		}

		parts = strings.Split(plugin, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid plugin name: %s", plugin)
		}

		plugin = parts[0]
		version := parts[1]

		if importPath != "" {
			plugin = importPath
		}

		pluginName := plugin
		parts = strings.Split(plugin, "/")
		if len(parts) > 3 {
			pluginName = strings.Join(parts[:3], "/")
		}

		pluginsData = append(pluginsData, Plugin{
			Hash:    "p" + hash,
			Name:    pluginName,
			Import:  plugin,
			Version: version,
		})
	}

	return pluginsData, nil
}

func createGoCode(build string, pluginsData []Plugin) error {
	f, err := os.OpenFile(build+"/plugin.go", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open plugin.go: %s", err)
	}
	defer f.Close()

	log.Debug().Msgf("Writing plugin.go")

	err = t.Execute(f, pluginsData)
	if err != nil {
		return fmt.Errorf("failed to write plugin.go: %s", err)
	}

	return nil
}

func BuildSO(plugins []string) (string, error) {
	path, err := buildSOBulk(plugins)
	if err == nil {
		return path, nil
	}

	return buildSODetails(plugins)
}

func buildSOBulk(plugins []string) (string, error) {
	build := initDir()

	log.Debug().Str("directory", build).Msgf("Building plugins")

	pluginsData, err := getPluginsData(plugins)
	if err != nil {
		return "", err
	}

	pluginsVersionned := []string{"get"}
	for _, plugin := range pluginsData {
		pluginsVersionned = append(pluginsVersionned, plugin.Name+"@"+plugin.Version)
	}

	cmdGetAll := exec.Command("go", pluginsVersionned...)
	cmdGetAll.Dir = build
	cmdGetAllOutput, err := cmdGetAll.CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Errorf("failed to run go get: %s %s", cmdGetAllOutput, err))
	}

	err = buildPlugin(build, pluginsData)
	if err != nil {
		return "", err
	}

	return build + "/plugin.so", nil
}

func buildSODetails(plugins []string) (string, error) {
	build := initDir()

	pluginsData, err := getPluginsData(plugins)
	if err != nil {
		return "", err
	}

	pluginsOK := []string{}
	failedPlugins := make([]string, 0)

	pluginsDataNew := []Plugin{}
	for _, plugin := range pluginsData {
		log.Debug().Msgf("Get plugin: %v", plugin)
		cmdTidy := exec.Command("go", "get", plugin.Import+"@"+plugin.Version)
		cmdTidy.Dir = build
		cmdTidyOutput, err := cmdTidy.CombinedOutput()
		if err != nil {
			fmt.Println(fmt.Errorf("failed to run go get: %s %s", cmdTidyOutput, err))
			failedPlugins = append(failedPlugins, fmt.Sprintf("failed to run go get: %s %s", cmdTidyOutput, err))
			continue
		}
		pluginsOK = append(pluginsOK, plugin.Name+"@"+plugin.Version)
		pluginsDataNew = append(pluginsDataNew, plugin)
	}

	err = buildPlugin(build, pluginsData)
	if err != nil {
		return "", err
	}

	return build + "/plugin.so", nil

}

func ReportErrors(plugins []string) string {
	build := initDir()

	pluginsData, err := getPluginsData(plugins)
	if err != nil {
		return err.Error()
	}

	failedPlugins := make([]string, 0)
	for _, plugin := range pluginsData {
		log.Debug().Msgf("Get plugin: %v", plugin)
		cmdTidy := exec.Command("go", "get", plugin.Name+"@"+plugin.Version)
		cmdTidy.Dir = build
		cmdTidyOutput, err := cmdTidy.CombinedOutput()
		if err != nil {
			fmt.Println(fmt.Errorf("failed to run go get: %s %s", cmdTidyOutput, err))
			failedPlugins = append(failedPlugins, fmt.Sprintf("failed to run go get: %s %s", cmdTidyOutput, err))
			continue
		}
	}

	return strings.Join(failedPlugins, "\n")
}
