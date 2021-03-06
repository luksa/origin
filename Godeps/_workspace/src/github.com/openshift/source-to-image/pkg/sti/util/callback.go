package util

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CallbackInvoker posts results to a callback URL when a STI build is done.
type CallbackInvoker interface {
	ExecuteCallback(callbackUrl string, success bool, messages []string) []string
}

// NewCallbackInvoker creates an instance of the default CallbackInvoker implementation
func NewCallbackInvoker() CallbackInvoker {
	invoker := &callbackInvoker{}
	invoker.postFunc = invoker.httpPost
	return invoker
}

type callbackInvoker struct {
	postFunc func(url, contentType string, body io.Reader) (resp *http.Response, err error)
}

// ExecuteCallback prepares a JSON payload and posts it to the specified callback URL
func (c *callbackInvoker) ExecuteCallback(callbackUrl string, success bool, messages []string) []string {
	buf := new(bytes.Buffer)
	writer := bufio.NewWriter(buf)
	for _, message := range messages {
		fmt.Fprintln(writer, message)
	}
	writer.Flush()

	d := map[string]interface{}{
		"payload": buf.String(),
		"success": success,
	}

	jsonBuffer := new(bytes.Buffer)
	writer = bufio.NewWriter(jsonBuffer)
	jsonWriter := json.NewEncoder(writer)
	jsonWriter.Encode(d)
	writer.Flush()

	var (
		resp *http.Response
		err  error
	)
	for retries := 0; retries < 3; retries++ {
		resp, err = c.postFunc(callbackUrl, "application/json", jsonBuffer)
		if err != nil {
			errorMessage := fmt.Sprintf("Unable to invoke callback: %v", err)
			messages = append(messages, errorMessage)
		}
		if resp != nil {
			if resp.StatusCode >= 300 {
				errorMessage := fmt.Sprintf("Callback returned with error code: %d", resp.StatusCode)
				messages = append(messages, errorMessage)
			}
			break
		}
	}
	return messages
}

func (*callbackInvoker) httpPost(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	return http.Post(url, contentType, body)
}
