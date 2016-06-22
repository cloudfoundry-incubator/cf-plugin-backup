package util

// Source: https://github.com/krujos/download_droplet_plugin // Apache 2.0 License

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/cloudfoundry/cli/plugin"
)

//CFDownloader real implementation to download droplets.
type CFDownloader struct {
	Cli    plugin.CliConnection
	Writer FileWriter
}

//Downloader interaface for implementing downloaders.
type Downloader interface {
	GetDroplet(guid string) ([]byte, error)
	SaveDropletToFile(filePath string, data []byte) error
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

func (downloader *CFDownloader) makeHTTPClient() (*http.Client, error) {
	sslDisabled, err := downloader.Cli.IsSSLDisabled()
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
func (downloader *CFDownloader) GetDroplet(guid string) ([]byte, error) {
	token, err := downloader.Cli.AccessToken()
	if nil != err {
		log.Fatal(err)
	}
	api, err := downloader.Cli.ApiEndpoint()
	if nil != err {
		log.Fatal(err)
	}
	client, err := downloader.makeHTTPClient()
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
func (downloader *CFDownloader) SaveDropletToFile(filePath string, data []byte) error {
	return downloader.Writer.WriteFile(filePath, data, 0644)
}

//Droplet interface
type Droplet interface {
	SaveDroplet(name string, path string) error
	GetDownloader() *Downloader
}

//CFDroplet utility for saving and whatnot.
type CFDroplet struct {
	Cli        plugin.CliConnection
	Downloader Downloader
}

//NewCFDroplet builds a new CF droplet
func NewCFDroplet(cli plugin.CliConnection, downloader Downloader) *CFDroplet {
	return &CFDroplet{
		Cli:        cli,
		Downloader: downloader,
	}
}

//SaveDroplet to the local filesystem.
func (d *CFDroplet) SaveDroplet(guid string, path string) error {
	data, err := d.Downloader.GetDroplet(guid)
	if nil != err {
		return err
	}
	err = d.Downloader.SaveDropletToFile(path, data)
	if nil != err {
		return err
	}
	return nil
}

func (d *CFDroplet) getGUID(appName string) (string, error) {
	app, err := d.Cli.GetApp(appName)
	return app.Guid, err
}

//GetDownloader attached to this dropplet.
func (d *CFDroplet) GetDownloader() *Downloader {
	return &d.Downloader
}
