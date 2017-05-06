package responses

import (
	"crypto/md5"
	"fmt"
	"github.com/tednaleid/ganda/execcontext"
	"github.com/tednaleid/ganda/logger"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sync"
)

func StartResponseWorkers(responses <-chan *http.Response, context *execcontext.Context) *sync.WaitGroup {
	var responseWaitGroup sync.WaitGroup
	responseWaitGroup.Add(context.ResponseWorkers)

	for i := 1; i <= context.ResponseWorkers; i++ {
		go func() {
			if context.WriteFiles {
				responseSavingWorker(responses, context)
			} else {
				responsePrintingWorker(responses, context)
			}
			responseWaitGroup.Done()
		}()
	}

	return &responseWaitGroup
}

func responseSavingWorker(responses <-chan *http.Response, context *execcontext.Context) {
	specialCharactersRegexp := regexp.MustCompile("[^A-Za-z0-9]+")

	responseWorker(responses, context.Logger, func(response *http.Response, body []byte) {
		filename := specialCharactersRegexp.ReplaceAllString(response.Request.URL.String(), "-")
		fullPath := saveBodyToFile(context.BaseDirectory, context.SubdirLength, filename, body)
		context.Logger.LogResponse(response.StatusCode, response.Request.URL.String()+" -> "+fullPath)
	})
}

func responsePrintingWorker(responses <-chan *http.Response, context *execcontext.Context) {
	responseWorker(responses, context.Logger, func(response *http.Response, body []byte) {
		context.Logger.LogResponse(response.StatusCode, response.Request.URL.String())
		if len(body) > 0 {
			context.Out.Printf("%s", body)
		}
	})
}

func responseWorker(responses <-chan *http.Response, logger *logger.LeveledLogger, responseBodyAction func(*http.Response, []byte)) {
	for response := range responses {
		body, err := ioutil.ReadAll(response.Body)
		response.Body.Close()

		if err != nil {
			logger.Warn("%s Response error status (%d): %v\n", response.Request.URL, response.StatusCode, err)
		} else {
			responseBodyAction(response, body)
		}
	}

}

func saveBodyToFile(baseDirectory string, subdirLength int, filename string, body []byte) string {
	directory := directoryForFile(baseDirectory, filename, subdirLength)
	fullPath := directory + filename
	err := ioutil.WriteFile(fullPath, body, 0644)

	if err != nil {
		panic(err)
	}

	return fullPath
}

func directoryForFile(baseDirectory string, filename string, subdirLength int) string {
	var directory string
	if subdirLength <= 0 {
		directory = fmt.Sprintf("%s/", baseDirectory)
	} else {
		sliceEnd := 1

		// don't create directories longer than 4 binary hex characters (4^16 = 65k directories)
		if subdirLength > 2 {
			sliceEnd = 2
		}

		md5val := md5.Sum([]byte(filename))
		directory = fmt.Sprintf("%s/%x/", baseDirectory, md5val[0:sliceEnd])
	}

	os.MkdirAll(directory, os.ModePerm)
	return directory
}
