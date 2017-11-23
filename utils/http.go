package utils

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

/*
   Get
*/
func HttpGet(path string, params map[string]string) (rsp []byte, err error) {

	u, _ := url.Parse(path)
	q := u.Query()

	if params != nil {
		for k, v := range params {
			q.Set(k, v)
		}
	}

	u.RawQuery = q.Encode()
	res, err := http.Get(u.String())
	if err != nil {
		log.Fatal(err)
		return
	}

	rsp, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
		return
	}

	statusCode := res.StatusCode
	// LogDebug(HTTP_CLIENT_TAG, "Status:", statusCode)

	if statusCode != 200 {
		err = errors.New("Invalid status code")
	}

	return
}

func HttpPostJson(path string, json []byte) (rsp []byte, err error) {

	body := bytes.NewBuffer(json)
	res, err := http.Post(path, "application/json;charset=utf-8", body)
	if err != nil {
		log.Fatal(err)
		return
	}
	rsp, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
		return
	}

	return
}

func HttpDownload(link string, path string) (err error) {

	base := filepath.Base(link)
	file := path + base
	println("written: ", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	stat, err := f.Stat() //获取文件状态
	if err != nil {
		return
	}
	defer f.Close()

	req, _ := http.NewRequest("GET", link, nil)
	req.Header.Set("Range", "bytes="+strconv.FormatInt(stat.Size(), 10)+"-")
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	if rsp.StatusCode != 200 {
		err = errors.New("Download File is not found")
		return
	}

	written, err := io.Copy(f, rsp.Body)
	if err != nil {
		return
	}

	println("size: ", written)

	return
}
