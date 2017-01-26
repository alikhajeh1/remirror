package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

var (
	http_client = http.Client{}
)

var mirrors = map[string]string{
	"/archlinux/":   "https://mirrors.kernel.org",
	"/centos/":      "https://mirrors.xmission.com",
	"/fedora/":      "https://mirrors.xmission.com",
	"/fedora-epel/": "https://mirrors.xmission.com",
	"/experticity/": "http://yum",
	"/java/":        "http://yum",
}

func should_cache(path string) bool {
	if strings.HasSuffix(path, ".pkg.tar.xz") {
		return true
	}
	if strings.HasSuffix(path, ".rpm") {
		return true
	}
	if strings.HasSuffix(path, "-rpm.bin") {
		return true
	}
	if strings.Contains(path, "/repodata/") && (strings.HasSuffix(path, ".gz") ||
		strings.HasSuffix(path, ".bz2") || strings.HasSuffix(path, ".xz")) {
		return true
	}
	return false
}

func main() {

	var (
		listen string
		data   string
	)

	flag.StringVar(&listen, "listen", ":80", "HTTP listen address")
	flag.StringVar(&data, "data", "/var/remirror", "Data storage path (data in here is public)")

	flag.Parse()

	fileserver := http.FileServer(http.Dir(data))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		log.Println(r.Method + " http://" + r.Host + r.RequestURI)

		err := func() error {

			upstream := ""

			for prefix, mirror := range mirrors {
				if strings.HasPrefix(r.URL.Path, prefix) {
					upstream = mirror
				}
			}

			if upstream == "" {
				fmt.Println("no upstream found for url", r.URL.Path)
				return HTTPError(404)
			}

			local_path := ""

			if should_cache(r.URL.Path) {
				local_path = data + path.Clean(r.URL.Path)

				_, err := os.Stat(local_path)
				if err == nil {
					fileserver.ServeHTTP(w, r)
					return nil
				}
			}

			log.Println("-->", upstream+r.RequestURI)

			req, err := http.NewRequest("GET", upstream+r.RequestURI, nil)
			if err != nil {
				return err
			}

			for k, vs := range r.Header {
				if !hopHeaders[k] {
					for _, v := range vs {
						req.Header.Add(k, v)
					}
				}
			}

			resp, err := http_client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			out := io.Writer(w)

			tmp_path := ""

			if resp.StatusCode == 200 && local_path != "" {
				tmp, err := ioutil.TempFile(data, "remirror_tmp_")
				if err != nil {
					return err
				}
				tmp_path = tmp.Name()
				//fmt.Println("tmp", tmp_path)

				defer tmp.Close()
				defer os.Remove(tmp_path)

				out = io.MultiWriter(out, tmp)
			}

			for k, vs := range resp.Header {
				if k == "Accept-Ranges" {
					continue
				}
				for _, v := range vs {
					//fmt.Printf("proxy back header %#v\t%#v\n", k, v)
					w.Header().Add(k, v)
				}
			}

			w.Header().Set("Server", "remirror")
			w.WriteHeader(resp.StatusCode)

			n, err := io.Copy(out, resp.Body)
			if err != nil {
				log.Println(err)
				return nil
			}

			if n != resp.ContentLength {
				if resp.ContentLength != -1 {
					log.Printf("Short data returned from server (Content-Length %d received %d)\n", resp.ContentLength, n)
				}
				// Not really an HTTP error, leave it up to the client
				return nil
			}

			if tmp_path != "" {
				os.MkdirAll(path.Dir(local_path), 0755)

				err = os.Rename(tmp_path, local_path)
				if err != nil {
					log.Println(err)
					return nil
				}
				log.Println(">:)")
			}

			return nil
		}()

		he, ok := err.(HTTPError)
		if ok {
			http.Error(w, he.Error(), he.Code())
			fmt.Println("\t\t", he.Error())
		} else if err != nil {
			http.Error(w, err.Error(), 500)
			fmt.Println("\t\t500 " + err.Error())
		}
	})

	log.Println("arch/fedora/centos/experticity mirror proxy listening on HTTP " + listen)
	log.Fatal(http.ListenAndServe(listen, nil))
}
