package util

// Source: https://github.com/krujos/download_droplet_plugin // Apache 2.0 License

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/cli/plugin"
)

//CFPackager real implementation to download droplets.
type CFPackager struct {
	Cli    plugin.CliConnection
	Writer FileWriter
	Reader FileReader
}

//Packager interaface for implementing downloaders.
type Packager interface {
	GetDroplet(guid string) ([]byte, error)
	SaveDropletToFile(filePath string, data []byte) error
	UploadDroplet(guid, path string) error
}

//FileWriter test shim for writing to a file.
type FileWriter interface {
	WriteFile(filename string, data []byte, perm os.FileMode) error
}

//CFFileWriter is a wrapper for ioutil.WriteFile
type CFFileWriter struct {
}

//WriteFile to disk
func (fw *CFFileWriter) WriteFile(filename string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(filename, data, perm)
}

//FileReader test shim for reading a file.
type FileReader interface {
	ReadFile(filename string) ([]byte, error)
}

//CFFileReader is a wrapper for ioutil.ReadFile
type CFFileReader struct {
}

//ReadFile reads a file
func (fr *CFFileReader) ReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}

func (packager *CFPackager) makeHTTPClient() (*http.Client, error) {
	sslDisabled, err := packager.Cli.IsSSLDisabled()
	if nil != err {
		return nil, err
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: sslDisabled}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}
	return client, nil

}

//GetDroplet from CF
func (packager *CFPackager) GetDroplet(guid string) ([]byte, error) {
	token, err := packager.Cli.AccessToken()
	if nil != err {
		log.Fatal(err)
	}
	api, err := packager.Cli.ApiEndpoint()
	if nil != err {
		log.Fatal(err)
	}
	client, err := packager.makeHTTPClient()
	if nil != err {
		log.Fatal(err)
	}
	url := api + "/v2/apps/" + guid + "/download"
	req, err := http.NewRequest("GET", url, nil)
	if nil != err {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", token)
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if nil != err {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Download failed. Status Code: %v. Body: %v", resp.Status, string(body))
	}

	return body, err
}

//SaveDropletToFile writes a downloaded droplet to file
func (packager *CFPackager) SaveDropletToFile(filePath string, data []byte) error {
	return packager.Writer.WriteFile(filePath, data, 0644)
}

//Droplet interface
type Droplet interface {
	SaveDroplet(name string, path string) error
	GetPackager() *Packager
	UploadDroplet(name, path string) error
}

//CFDroplet utility for saving and whatnot.
type CFDroplet struct {
	Cli        plugin.CliConnection
	Packager   Packager
	HTTPClient *http.Client
}

//NewCFDroplet builds a new CF droplet
func NewCFDroplet(cli plugin.CliConnection, packager Packager) *CFDroplet {

	return &CFDroplet{
		Cli:      cli,
		Packager: packager,
	}
}

//SaveDroplet to the local filesystem.
func (d *CFDroplet) SaveDroplet(guid string, path string) error {
	data, err := d.Packager.GetDroplet(guid)
	if nil != err {
		return err
	}
	err = d.Packager.SaveDropletToFile(path, data)
	if nil != err {
		return err
	}
	return nil
}

// UploadDroplet uploads an apps droplet
func (d *CFDroplet) UploadDroplet(guid, path string) error {
	return d.Packager.UploadDroplet(guid, path)
}

// UploadDroplet uploads an apps droplet
func (packager *CFPackager) UploadDroplet(guid, path string) error {
	token, err := packager.Cli.AccessToken()
	if nil != err {
		log.Fatal(err)
	}
	api, err := packager.Cli.ApiEndpoint()
	if nil != err {
		log.Fatal(err)
	}
	client, err := packager.makeHTTPClient()
	if nil != err {
		log.Fatal(err)
	}

	data, err := packager.Reader.ReadFile(path)
	if err != nil {
		return err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	err = writer.WriteField("resources", "[]")
	if err != nil {
		return err
	}
	err = writer.WriteField("application", "[]")

	part, err := writer.CreateFormFile("application", filepath.Base(path))
	if err != nil {
		return err
	}
	_, err = part.Write(data)

	err = writer.Close()
	if err != nil {
		return err
	}

	uri := fmt.Sprintf("%s/v2/apps/%s/bits", api, guid)

	request, err := http.NewRequest("PUT", uri, body)
	if err != nil {
		return err
	}

	request.Header.Add("Content-Type", writer.FormDataContentType())
	request.Header.Add("Authorization", token)

	resp, err := client.Do(request)

	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Received %d while uploading file for app %s", resp.StatusCode, guid)
	}

	return nil
}

func (d *CFDroplet) getGUID(appName string) (string, error) {
	app, err := d.Cli.GetApp(appName)
	return app.Guid, err
}

//GetPackager attached to this dropplet.
func (d *CFDroplet) GetPackager() *Packager {
	return &d.Packager
}
