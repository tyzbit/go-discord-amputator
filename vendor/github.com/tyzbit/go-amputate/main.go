package goamputate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	amputatorApi string = "https://www.amputatorbot.com/api/v1"
	userAgent    string = "github.com/tyzbit/go-amputate"
	gac          string = "true"
	md           string = "3"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

// fillOptionsDefaults takes AmputationRequestOptions and fills in defaults
func fillOptionsDefaults(o map[string]string) {
	if o["gac"] == "" {
		o["gac"] = gac
	}
	if o["gac"] == "" {
		o["gac"] = md
	}
}

// Amputate takes a slice of strings of URLs and returns the amputated versions
// of the URLs. Not guaranteed to return the same number of values.
//
// Current options:
// gac: Guess and Check, if the canonical URL can't be determined, try guessing
// md: Max depth to follow links in order to determine canonical URL
func Amputate(urls []string, o map[string]string) ([]string, error) {
	fillOptionsDefaults(o)
	ampRequest := AmputationRequest{
		options: o,
		urls:    urls,
	}
	ampResponse, err := Convert(ampRequest)
	if err != nil {
		return nil, err
	}

	ampUrls, err := GetCanonicalUrls(ampResponse)
	if err != nil {
		return nil, err
	}

	return ampUrls, nil
}

// Convert takes an AmputationRequest and returns a byte slice of the response
// from the AmputatorAPI. External callers should probably use Amputate()
// instead.
func Convert(r AmputationRequest) ([]byte, error) {
	options := ""
	for option, value := range r.options {
		options = fmt.Sprintf("%v=%v", option, value)
	}
	url := fmt.Sprintf("%v/convert?%v&q=%v", amputatorApi, options, strings.Join(r.urls, ";"))
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", userAgent)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// GetCanonicalUrls takes a byte slice of an Amputator API return object and
// returns a slice of strings of unique non_amp URLs.
func GetCanonicalUrls(body []byte) ([]string, error) {
	urls := []string{}

	ampResponse := []AmputationResponseObject{}
	err := json.Unmarshal([]byte(body), &ampResponse)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal json: %v, err: %v", string(body), err)
	}

	for _, ampObject := range ampResponse {
		for _, canonical := range ampObject.Canonicals {
			if !canonical.IsAmp {
				urls = append(urls, canonical.Url)
			}
		}
	}

	uniqueUrls := removeDuplicateValues(urls)
	return uniqueUrls, nil
}

func removeDuplicateValues(strings []string) []string {
	keys := make(map[string]bool)
	list := []string{}

	// If the key(values of the slice) is not equal
	// to the already present value in new slice (list)
	// then we append it. else we jump on another element.
	for _, entry := range strings {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
