package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/spacesedan/testing-course/webapp/pkg/data"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"testing"
)

func Test_application_handlers(t *testing.T) {
	tests := []struct {
		name                    string
		url                     string
		expectedStatusCode      int
		expectedFirstStatusCode int
		expectedUrl             string
	}{
		{"home", "/", http.StatusOK, http.StatusOK, "/"},
		{"404", "/fish", http.StatusNotFound, http.StatusNotFound, "/fish"},
		{"profile", "/user/profile", http.StatusOK, http.StatusTemporaryRedirect, "/"},
	}

	routes := app.routes()

	// create a test server
	ts := httptest.NewTLSServer(routes)
	defer ts.Close()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// range through the test table

	for _, e := range tests {
		resp, err := ts.Client().Get(ts.URL + e.url)
		if err != nil {
			t.Log(err)
			t.Fatal(err)
		}

		if resp.StatusCode != e.expectedStatusCode {
			t.Errorf("for %s: expected status %d, but got %d", e.name, e.expectedStatusCode, resp.StatusCode)
		}

		if resp.Request.URL.Path != e.expectedUrl {
			t.Errorf("%s: expected final url of %s, but got %s", e.name, e.expectedUrl, resp.Request.URL.Path)
		}

		resp2, _ := client.Get(ts.URL + e.url)
		if resp2.StatusCode != e.expectedFirstStatusCode {
			t.Errorf("%s: expected first returned status code to be %d, but got %d", e.name, e.expectedFirstStatusCode, resp2.StatusCode)
		}

	}

}

func TestApplication_Home(t *testing.T) {
	tests := []struct {
		name         string
		putInSession string
		expectedHTML string
	}{
		{"first visit", "", "<small>From Session: "},
		{"second visit", "test", "<small>From Session: test"},
	}

	for _, e := range tests {
		// create request
		req, _ := http.NewRequest(http.MethodGet, "/", nil)

		req = addContextAndSessionToRequest(req, app)
		_ = app.Session.Destroy(req.Context())

		// put something in session when we are not testing an empty session string
		if e.putInSession != "" {
			app.Session.Put(req.Context(), "test", e.putInSession)
		}

		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(app.Home)

		handler.ServeHTTP(rr, req)

		// check status code
		if rr.Code != http.StatusOK {
			t.Errorf("TestApplication_Home returned wrong status code; expected %d but got %d", http.StatusOK, rr.Code)
		}

		body, _ := io.ReadAll(rr.Body)
		if !strings.Contains(string(body), e.expectedHTML) {
			t.Errorf("%s: did not find %s in response body", e.name, e.expectedHTML)
		}
	}
}

func TestApplication_renderWithBadTemplate(t *testing.T) {
	// set template to a location with a bad template
	pathToTemplates = "./testdata/"

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req = addContextAndSessionToRequest(req, app)

	rr := httptest.NewRecorder()

	err := app.render(rr, req, "bad.page.gohtml", &TemplateData{})
	if err == nil {
		t.Error("expected error from bad template but did not get one")
	}

	pathToTemplates = "./../../templates/"
}

func getCtx(r *http.Request) context.Context {
	return context.WithValue(r.Context(), contextUserKey, "unknown")
}

func addContextAndSessionToRequest(req *http.Request, app application) *http.Request {
	req = req.WithContext(getCtx(req))

	ctx, _ := app.Session.Load(req.Context(), req.Header.Get("X-Session"))

	return req.WithContext(ctx)
}

func TestApplication_Login(t *testing.T) {
	tests := []struct {
		name               string
		postedData         url.Values
		expectedStatusCode int
		expectedLoc        string
	}{
		{
			name: "valid login",
			postedData: url.Values{
				"email":    {"admin@example.com"},
				"password": {"secret"},
			},
			expectedStatusCode: http.StatusSeeOther,
			expectedLoc:        "/user/profile",
		}, {
			name: "missing form data",
			postedData: url.Values{
				"email":    {""},
				"password": {""},
			},
			expectedStatusCode: http.StatusSeeOther,
			expectedLoc:        "/",
		}, {
			name: "user not found",
			postedData: url.Values{
				"email":    {"WRONG@EMAIL.com"},
				"password": {"WRONG_PASSWORD"},
			},
			expectedStatusCode: http.StatusSeeOther,
			expectedLoc:        "/",
		}, {
			name: "",
			postedData: url.Values{
				"email":    {"admin@example.com"},
				"password": {"WRONG_PASSWORD"},
			},
			expectedStatusCode: http.StatusSeeOther,
			expectedLoc:        "/",
		},
	}

	for _, e := range tests {
		// setup the login request
		req, _ := http.NewRequest(http.MethodPost, "/login", strings.NewReader(e.postedData.Encode()))
		req = addContextAndSessionToRequest(req, app)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		// create a test response recorder
		rr := httptest.NewRecorder()
		// handler the request
		handler := http.HandlerFunc(app.Login)
		handler.ServeHTTP(rr, req)

		if rr.Code != e.expectedStatusCode {
			t.Errorf("%s: returned a wrong status code; expected %d, but got %d", e.name, e.expectedStatusCode, rr.Code)
		}

		// get the location of the response
		actualLoc, err := rr.Result().Location()
		if err == nil {
			if actualLoc.String() != e.expectedLoc {
				t.Errorf("%s: returned wrong expected location; expected %s, but got %s", e.name, e.expectedLoc, actualLoc)
			}
		} else {
			t.Errorf("%s: no location header set", e.name)
		}

	}
}

func TestApplication_UploadFiles(t *testing.T) {
	// set up pipes
	pr, pw := io.Pipe()

	// create a new writer, of type *io.Writer
	writer := multipart.NewWriter(pw)

	// create a wait group, adn add 1 to it
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// simulate uploading a file using a goroutine and a writer
	go simulateJPGUpload("./testdata/img.JPG", writer, t, wg)

	// read from the pipe which receives data
	request := httptest.NewRequest(http.MethodPost, "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	// call app.UploadFiles
	uploadedFiles, err := app.UploadFiles(request, "./testdata/uploads/")
	if err != nil {
		t.Error(err)
	}

	//preform our tests
	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].OriginalFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist :%s", err.Error())

	}
	// clean up
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].OriginalFileName))

	wg.Wait()
}

func simulateJPGUpload(fileToUpload string, writer *multipart.Writer, t *testing.T, wg *sync.WaitGroup) {
	defer writer.Close()
	defer wg.Done()

	// create the form data filed 'file' with value being filename
	part, err := writer.CreateFormFile("file", path.Base(fileToUpload))
	if err != nil {
		t.Error(err)
	}

	// open the actual file
	f, err := os.Open(fileToUpload)
	if err != nil {
		t.Error(err)
	}
	defer f.Close()

	// decode the image
	img, _, err := image.Decode(f)
	if err != nil {
		t.Error("error decoding image")
	}

	// write the jpg to io.Writer
	err = jpeg.Encode(part, img, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestApplication_UploadProfilePic(t *testing.T) {
	pathToStatic = "./testdata/uploads/"
	filepath := "./testdata/img.JPG"

	// specify a field name for the form
	fieldName := "file"

	// create a bytes.Buffer to act as the request body
	body := new(bytes.Buffer)

	// create a new writer
	mw := multipart.NewWriter(body)

	file, err := os.Open(filepath)
	if err != nil {
		t.Fatal(err)
	}

	w, err := mw.CreateFormFile(fieldName, filepath)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = io.Copy(w, file); err != nil {
		t.Fatal(err)
	}

	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req = addContextAndSessionToRequest(req, app)
	app.Session.Put(req.Context(), "user", data.User{ID: 1})
	req.Header.Add("Content-Type", mw.FormDataContentType())

	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(app.UploadProfilePic)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("wrong status code")
	}

	_ = os.Remove("./testdata/uploads/img.JPG")
}
