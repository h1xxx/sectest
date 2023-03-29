package main

import (
	"fmt"
	"sync"

	str "strings"
)

func (t *targetT) whatWeb(host, scan string, pi portInfoT, wg *sync.WaitGroup) {
	var c cmdT
	c.name = fmt.Sprintf("%s_%d", scan, pi.port)
	c.bin = "whatweb"

	var level int
	switch scan {
	case "fast":
		level = 3
	case "full":
		level = 4
	}

	var sslSuffix string
	if pi.tunnel == "ssl" {
		sslSuffix = "s"
	}

	argsS := fmt.Sprintf("-a%d -t64 --colour=never -v --no-errors", level)

	c.args = str.Split(argsS, " ")

	c.args = append(c.args, "-U")
	c.args = append(c.args, getRandomUA())

	c.args = append(c.args, fmt.Sprintf("http%s://%s:%d",
		sslSuffix, host, pi.port))

	runCmd(host, &c)
	wg.Done()
}