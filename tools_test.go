package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools
	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Error("RandomString did not return a string of the correct length")
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{name: "Allowed no rename", allowedTypes: []string{"image/png", "image/jpeg"}, renameFile: false, errorExpected: false},
	{name: "Allowed rename", allowedTypes: []string{"image/png", "image/jpeg"}, renameFile: true, errorExpected: false},
	{name: "not allowed", allowedTypes: []string{"image/jpeg"}, renameFile: false, errorExpected: true},
}

func TestTools_UploadFile(t *testing.T) {
	for _, e := range uploadTests {
		// setup a pipe to avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			// create the form data fieled "file"
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error("Error creating form file:", err)
				return
			}

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error("Error opening file:", err)
				return
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("Error decoding image:", err)
				return
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error("Error encoding image:", err)
				return
			}
		}()

		// read from the pipe which receives data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes

		UploadedFiles, err := testTools.UploadFile(request, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error was not expected but got one: %v", e.name, err)
		}
		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", UploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to be uploaded but it was not found: %s", e.name, err.Error())

			}
			// clean up
			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", UploadedFiles[0].NewFileName))
		}
		if !e.errorExpected && err != nil {
			t.Errorf("%s: error was not expected but got one: %v", e.name, err)
		}
		wg.Wait()
	}
}

func TestTools_UploadSingleFile(t *testing.T) {
	for _, e := range uploadTests {
		// setup a pipe to avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)

		go func() {
			defer writer.Close()

			// create the form data fieled "file"
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error("Error creating form file:", err)
				return
			}

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error("Error opening file:", err)
				return
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("Error decoding image:", err)
				return
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error("Error encoding image:", err)
				return
			}
		}()

		// read from the pipe which receives data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools

		UploadedFiles, err := testTools.UploadSingleFile(request, "./testdata/uploads/", true)
		if err != nil {
			t.Errorf("%s: error was not expected but got one: %v", e.name, err)
		}

		if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", UploadedFiles.NewFileName)); os.IsNotExist(err) {
			t.Errorf("expected file to be uploaded but it was not found: %s", err.Error())

		}
		// clean up
		_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", UploadedFiles.NewFileName))

	}
}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTools Tools
	err := testTools.CreateDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Error("Error creating directory:", err)
	}

	err = testTools.CreateDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Error("Error creating directory:", err)
	}

	_ = os.Remove("./testdata/myDir")
}

var slugTests = []struct {
	name          string
	s             string
	expectedSlug  string
	errorExpected bool
}{
	{name: "Normal string", s: "This is a test string to be slugified!", expectedSlug: "this-is-a-test-string-to-be-slugified", errorExpected: false},
	{name: "Empty string", s: "", expectedSlug: "", errorExpected: true},
	{name: "complex string", s: "Hello, World! This is a test. +  &^123", expectedSlug: "hello-world-this-is-a-test-123", errorExpected: false},
	{name: "japanese string", s: "こんにちは世界", expectedSlug: "", errorExpected: true},
	{name: "japanese string and roman characters", s: "hello world こんにちは世界", expectedSlug: "hello-world", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var testTools Tools

	for _, e := range slugTests {
		slug, err := testTools.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error was not expected but got one: %s", e.name, err.Error())
		}
		if !e.errorExpected && slug != e.expectedSlug {
			t.Errorf("%s: expected slug '%s' but got '%s'", e.name, e.expectedSlug, slug)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTools Tools

	testTools.DownloadStaticFile(rr, req, "./testdata/", "pic.jpg", "my-pic.jpg")

	res := rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "98827" {
		t.Error("wrong content length of", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"my-pic.jpg\"" {
		t.Error("wrong content disposition of", res.Header["Content-Disposition"][0])
	}

	_, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error("error reading response body:", err)
	}
}

var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{name: "good json", json: `{"foo": "bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: false},
	{name: "badly formatted json", json: `{"foo":`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "incorect type", json: `{"foo": 1}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "two json files", json: `{"foo": "1"}{"foo": "2"}`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "syntax error", json: `{"foo": 1"`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "unknown field", json: `{"fooo": "1"`, errorExpected: true, maxSize: 1024, allowUnknown: false},
	{name: "unknown fields in json", json: `{"fooo": "1"}`, errorExpected: false, maxSize: 1024, allowUnknown: true},
	{name: "missing field name", json: `{jack: "1"}`, errorExpected: true, maxSize: 1024, allowUnknown: true},
	{name: "file too large", json: `{"foo": "bar"}`, errorExpected: true, maxSize: 5, allowUnknown: true},
	{name: "not json", json: `hello world`, errorExpected: true, maxSize: 1024, allowUnknown: true},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTools Tools

	for _, e := range jsonTests {
		// set the max file size
		testTools.MaxJSONSize = e.maxSize

		// allow or disallow unknown fields
		testTools.AllowUnknownFields = e.allowUnknown

		// declare a variable to read the decoded JSON into
		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		// create a new HTTP request with the JSON string as the body
		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error", err)
		}

		// ceate a recorder
		rr := httptest.NewRecorder()

		err = testTools.ReadJSON(rr, req, &decodedJSON)

		if e.errorExpected && err == nil {
			t.Errorf("%s: expected an error but did not get one", e.name)
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: did not expect an error but got one: %s", e.name, err.Error())
		}

		req.Body.Close()
	}
}

func TestTools_WriteJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "Success",
	}

	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := testTools.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write JSON: %v", err)
	}
}

func TestTools_ErrorJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	err := testTools.ErrorJSON(rr, errors.New("this is an error"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}

	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Error("error decoding JSON response:", err)
	}

	if !payload.Error {
		t.Error("expected error to be true but got false")
	}

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status code %d but got %d", http.StatusServiceUnavailable, rr.Code)
	}
}
