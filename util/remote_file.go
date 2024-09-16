package util

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

type RemoteContentFile struct {
	Content string `json:"content"`
}

type RemoteContentFileURL struct {
	url string `json:"url"`
}

// FetchRemoteFile could is usable when you fetch github private repository file, or raw content file, etc.
func FetchRemoteFile(url, token string) ([]string, error) {

	req, _ := http.NewRequest("GET", url, nil)
	if token == "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("Failed to fetch file: %s", resp.Status))
	}

	var result []string

	body, _ := ioutil.ReadAll(resp.Body)

	var (
		file    RemoteContentFile
		content string
	)

	// it checks if there is "content" field and if there is, it'll use it as result.
	// or if not, it'll check if there is any "url" field and request it.
	// but even there are no "url" field, it'll use all of it as result.(ex. raw.github...)
	//
	// This feature is designed for github files
	// because github responses requested file through "content" field, and if requested path is not file just directory,
	// it responses urls which children files exists.
	err = json.Unmarshal(body, &file)

	if err != nil {
		var remoteUrls []RemoteContentFileURL
		err = json.Unmarshal(body, &remoteUrls)
		if err != nil {
			content = string(body)
		} else {
			for _, childContent := range remoteUrls {
				childContents, err := FetchRemoteFile(childContent.url, token)
				if err != nil {
					return nil, err
				}

				result = append(result, childContents...)
			}
			return result, nil
		}
	} else {
		content = file.Content
	}

	decodedContent, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return nil, err
	}
	result = append(result, string(decodedContent))

	return result, err
}
