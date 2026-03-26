package toolkit

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
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
