package core

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func externalHttpSourceToPath(lib string, url string) (path string) {
	home, _ := os.LookupEnv("HOME")
	localBase := filepath.Join(home, ".jokerd", "deps", strings.SplitN(url, "//", 2)[1])
	libBase := filepath.Join(strings.Split(lib, ".")...) + ".joke"
	libPath := filepath.Join(localBase, libBase)
	libPathDir := filepath.Dir(libPath)

	if _, err := os.Stat(libPathDir); os.IsNotExist(err) {
		os.MkdirAll(libPathDir, 0777)
	}

	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		if !strings.HasSuffix(url, ".joke") {
			url = url + libBase
		}
		resp, err := http.Get(url)
		PanicOnErr(err)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			panic(RT.NewError(fmt.Sprintf("Unable to retrieve: %s\nServer response: %d", url, resp.StatusCode)))
		}

		out, err := os.Create(libPath)
		defer out.Close()
		PanicOnErr(err)

		_, err = io.Copy(out, resp.Body)
		PanicOnErr(err)
	}

	return libPath
}

func externalSourceToPath(lib string, url string) (path string) {
	httpPath, _ := regexp.MatchString("http://|https://", url)
	if httpPath {
		return externalHttpSourceToPath(lib, url)
	} else {
		return filepath.Join(append([]string{url}, strings.Split(lib, ".")...)...) + ".joke"
	}
}
