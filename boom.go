// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	gourl "net/url"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/valyala/fasthttp"
	"log"
	"runtime/pprof"
)

const (
	headerRegexp = "^([\\w-]+):\\s*(.+)"
	authRegexp   = "^([\\w-\\.]+):(.+)"
)

var (
	m           = flag.String("m", "GET", "")
	headers     = flag.String("h", "", "")
	body        = flag.String("d", "", "")
	accept      = flag.String("A", "", "")
	contentType = flag.String("T", "text/html", "")

	output = flag.String("o", "", "")

	c = flag.Int("c", 50, "")
	n = flag.Int("n", 200, "")
	q = flag.Int("q", 0, "")
	t = flag.Int("t", 0, "")

	insecure           = flag.Bool("allow-insecure", false, "")
	disableCompression = flag.Bool("disable-compression", false, "")
	disableKeepAlives  = flag.Bool("disable-keepalive", false, "")
	proxyAddr          = flag.String("x", "", "")

	flagCPUProfile = flag.String("cpuprofile", "", "write cpu profile to file")
	flagMemProfile = flag.String("memprofile", "", "write memory profile to file")
)

var usage = `Usage: boom [options...] <url>

Options:
  -n  Number of requests to run.
  -c  Number of requests to run concurrently. Total number of requests cannot
      be smaller than the concurency level.
  -q  Rate limit, in seconds (QPS).
  -o  Output type. If none provided, a summary is printed.
      "csv" is the only supported alternative. Dumps the response
      metrics in comma-seperated values format.

  -m  HTTP method, one of GET, POST, PUT, DELETE, HEAD, OPTIONS.
  -h  Custom HTTP headers, name1:value1;name2:value2.
  -t  Timeout in ms.
  -A  HTTP Accept header.
  -d  HTTP request body.
  -T  Content-type, defaults to "text/html".
  -a  Basic authentication, username:password.
  -x  HTTP Proxy address as host:port.

  -allow-insecure       Allow bad/expired TLS/SSL certificates.
  -disable-compression  Disable compression.
  -disable-keepalive    Disable keep-alive, prevents re-use of TCP
                        connections between different HTTP requests.
`

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(usage, runtime.NumCPU()))
	}

	flag.Parse()
	if flag.NArg() < 1 {
		usageAndExit("")
	}

	if *flagCPUProfile != "" {
		log.Print("Profiling to file ", *flagCPUProfile, " started.")
		f, err := os.Create(*flagCPUProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	num := *n
	conc := *c
	q := *q

	if num <= 0 || conc <= 0 {
		usageAndExit("n and c cannot be smaller than 1.")
	}

	var (
		url, method string
		header      fasthttp.RequestHeader
	)

	url = flag.Args()[0]
	method = strings.ToUpper(*m)

	// set content-type
	header.SetContentType(*contentType)
	//	// set any other additional headers
	if *headers != "" {
		headers := strings.Split(*headers, ";")
		for _, h := range headers {
			match, err := parseInputWithRegexp(h, headerRegexp)
			if err != nil {
				usageAndExit(err.Error())
			}
			header.Set(match[1], match[2])
		}
	}

	if *accept != "" {
		header.Set("Accept", *accept)
	}

	var proxyURL *gourl.URL
	if *proxyAddr != "" {
		var err error
		proxyURL, err = gourl.Parse(*proxyAddr)
		if err != nil {
			usageAndExit(err.Error())
		}
	}

	header.SetMethod(method)
	header.SetRequestURI(url)
	if !*disableCompression {
		header.Set("Accept-Encoding", "gzip")
	}

	var req fasthttp.Request
	header.CopyTo(&req.Header)
	req.AppendBodyString(*body)

	(&Boomer{
		Request:            &req,
		N:                  num,
		C:                  conc,
		Qps:                q,
		Timeout:            *t,
		AllowInsecure:      *insecure,
		DisableCompression: *disableCompression,
		DisableKeepAlives:  *disableKeepAlives,
		ProxyAddr:          proxyURL,
		Output:             *output,
	}).Run()
}

func usageAndExit(msg string) {
	if msg != "" {
		fmt.Fprintf(os.Stderr, msg)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}

func parseInputWithRegexp(input, regx string) ([]string, error) {
	re := regexp.MustCompile(regx)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 1 {
		return nil, fmt.Errorf("could not parse the provided input; input = %v", input)
	}
	return matches, nil
}
