package main

import (
	"errors"
	"fmt"
	"os"
	"sync"

	fp "path/filepath"
	str "strings"

	"sectest/ffuf"
)

func (t *targetT) testHttp(port int, portInfo portInfoT) {
	print("testing %s on port %d...\n", portInfo.service, port)

	httpWg := &sync.WaitGroup{}
	httpWg.Add(2)
	go t.whatWeb(t.host, "fast", port, httpWg)
	go t.ffufCommon(t.host, "fast", port, httpWg)
	httpWg.Wait()

	httpWg.Add(1)
	go t.ffufRec(t.host, "fast", "1", port, httpWg)
	httpWg.Wait()

	httpWg.Add(2)
	go t.whatWeb(t.host, "full", port, httpWg)
	go t.ffufCommon(t.host, "full", port, httpWg)
	httpWg.Wait()

	httpWg.Add(1)
	go t.ffufRec(t.host, "full", "1", port, httpWg)
	httpWg.Wait()

	t.wg.Done()
}

func (t *targetT) whatWeb(host, scan string, port int, wg *sync.WaitGroup) {
	var c cmdT
	c.name = scan
	c.bin = "whatweb"

	var level int
	switch scan {
	case "fast":
		level = 3
	case "full":
		level = 4
	}

	argsS := fmt.Sprintf("-a%d -t64 --colour=never -v --no-errors", level)

	c.args = str.Split(argsS, " ")

	c.args = append(c.args, "-U")
	c.args = append(c.args, getRandomUA())

	c.args = append(c.args, fmt.Sprintf("%s:%d", host, port))

	runCmd(host, &c)
	wg.Done()
}

func (t *targetT) ffufCommon(host, scan string, port int, wg *sync.WaitGroup) {
	var c cmdT
	c.name = "common_" + scan
	c.bin = "ffuf"

	wordlist := "./data/http_common_" + scan

	f := "-noninteractive -r -t 64 -r -o %s/ffuf/common_%s.json "
	f += "-w %s:FUZZ -u http://%s:%d/FUZZ"
	argsS := fmt.Sprintf(f, t.host, scan, wordlist, host, port)

	c.args = str.Split(argsS, " ")

	c.args = append(c.args, "-H")
	c.args = append(c.args, fmt.Sprintf("User-Agent: %s", getRandomUA()))

	runCmd(host, &c)
	err := ffuf.CleanFfuf(fp.Join(host, c.bin, c.name))
	errExit(err)

	wg.Done()
}

// l - recursion level
func (t *targetT) ffufRec(host, scan, l string, port int, wg *sync.WaitGroup) {
	file := fmt.Sprintf("%s/ffuf/common_%s.json", t.host, scan)
	ffufRes, err := ffuf.GetUrls(file)
	errExit(err)

	err = ffuf.GetDirs(ffufRes, t.host, scan, l, "data/http_dir")
	errExit(err)

	var c cmdT
	c.name = "rec_" + scan + "_l" + l
	c.bin = "ffuf"

	dirlist := fp.Join(host, "tmp", "ffuf_rec_"+scan+"_l"+l)
	filelist := "./data/http_file_rec_" + scan

	_, err = os.Stat(dirlist)
	if errors.Is(err, os.ErrNotExist) {
		wg.Done()
		return
	}

	f := "-noninteractive -r -t 64 -r -o %s/ffuf/rec_%s_l%s.json "
	f += "-fc 403 "
	f += "-w %s:DIR -w %s:FILE -u http://%s:%d/DIR/FILE"
	argsS := fmt.Sprintf(f, t.host, scan, l, dirlist, filelist, host, port)

	c.args = str.Split(argsS, " ")

	c.args = append(c.args, "-H")
	c.args = append(c.args, fmt.Sprintf("User-Agent: %s", getRandomUA()))

	runCmd(host, &c)
	err = ffuf.CleanFfuf(fp.Join(host, c.bin, c.name))
	errExit(err)

	wg.Done()
}
