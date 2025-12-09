package utils

import (
	"bytes"
	"io"
	"net/http"
)

func Get(url string) (error, []byte) {
	res, err := http.Get(url)
	if err != nil {
		return err, nil
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	return err, body
}

// Post function to send data with POST method
func Post(url string, data []byte) (error, []byte) {
	return doRequest("POST", url, data)
}

// Put function to update data with PUT method
func Put(url string, data []byte) (error, []byte) {
	return doRequest("PUT", url, data)
}

// Patch function to update data with PATCH method
func Patch(url string, data []byte) (error, []byte) {
	return doRequest("PATCH", url, data)
}

// Delete function to delete data with DELETE method
func Delete(url string) (error, []byte) {
	return doRequest("DELETE", url, nil)
}

// Helper function for POST, PUT, PATCH, and DELETE requests
func doRequest(method, url string, data []byte) (error, []byte) {
	var reqBody io.Reader
	if data != nil {
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return err, nil
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err, nil
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	return err, body
}
