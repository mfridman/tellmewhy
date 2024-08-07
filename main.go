package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

const (
	goProxyURL = "https://proxy.golang.org"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("")
	if err := run("go.mod"); err != nil {
		log.Fatal(err)
	}
}

func run(filename string) error {
	modFile, err := openModFile(filename)
	if err != nil {
		return err
	}
	current := modFile.Go.Version

	for _, require := range modFile.Require {
		ok, err := hasNewVersion(require.Mod)
		if err != nil {
			return err
		}
		if ok {
			log.Printf("module %s has new version: %s", require.Mod.Path, require.Mod.Version)
		}
		if err := execGoGet("get", "-u", require.Mod.Path+"@latest"); err != nil {
			return err
		}
		updated, err := openModFile(filename)
		if err != nil {
			return err
		}
		if updated.Go.Version != current {
			return fmt.Errorf(out, updated.Go.Version, require.Mod.Path)
		}
	}
	return nil
}

var out = `Error: Go version mismatch

The required Go version has been updated to %s.
This change is due to an update in the module: %s`

func execGoGet(args ...string) error {
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func openModFile(filename string) (*modfile.File, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return modfile.Parse(filename, data, nil)
}

func hasNewVersion(mod module.Version) (bool, error) {
	client := http.DefaultClient
	url := fmt.Sprintf("%s/%s/@latest", goProxyURL, mod.Path)
	url = strings.ToLower(url)
	resp, err := client.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %d for %s", resp.StatusCode, url)
	}
	var data struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false, err
	}
	n := semver.Compare(data.Version, mod.Version)
	return n > 0, nil
}
