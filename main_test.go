package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

// make sure that vince started with configuration and calls kase after vince is
// up and running.
//
// This will make sure all resources are cleared/released before the function exits.
func runTest(t *testing.T, v *vinceConfiguration, kase ...func(context.Context, *testing.T)) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fmt.Println("starting tests")
	ready := make(chan bool)
	go func() {
		err := startEverything(ctx, v, func() {
			ready <- true
		})
		if err != nil {
			t.Log(err)
			ready <- false
		}
	}()
	ok := <-ready
	if !ok {
		b, _ := ioutil.ReadFile(v.confFile)
		fmt.Println(string(b))
		return
	}
	for _, f := range kase {
		f(ctx, t)
	}
}

type httpCheckFn func(context.Context, *testing.T, *http.Response)

func checkCode(code int) httpCheckFn {
	return func(ctx context.Context, t *testing.T, res *http.Response) {
		if res.StatusCode != code {
			t.Errorf("check code: expected %d got %d", code, res.StatusCode)
		}
	}
}

func checkHeader(name, value string) httpCheckFn {
	return func(ctx context.Context, t *testing.T, res *http.Response) {
		if res.Header.Get(name) != value {
			t.Errorf("check header %q: expected %s got %s", name, value, res.Header.Get(name))
		}
	}
}

func checkBody(file string) httpCheckFn {
	return func(ctx context.Context, t *testing.T, res *http.Response) {
		f, err := ioutil.ReadFile(file)
		if err != nil {
			t.Error(err)
			return
		}
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(b, f) {
			t.Errorf("check  body: \n expected %s \ngot \n%s", string(f), string(b))
		}
	}
}

func runHTTP(ctx context.Context, t *testing.T, r *http.Request, checks ...httpCheckFn) {
	t.Helper()
	res, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Error(err)
		return
	}
	defer res.Body.Close()
	for _, f := range checks {
		f(ctx, t, res)
	}
}

func setup(file string) (*vinceConfiguration, func(), error) {
	dir, err := ioutil.TempDir("", "vince-test-suite")
	if err != nil {
		return nil, nil, err
	}
	if strings.Contains(file, "{{") {
		m := funcMap()
		m["test_http_globals"] = testHTTPpGlobals
		tpl, err := template.New("vince").Funcs(m).Parse(file)
		if err != nil {
			return nil, nil, err
		}
		var buf bytes.Buffer
		err = tpl.Execute(&buf, map[string]string{"dir": dir})
		if err != nil {
			return nil, nil, err
		}
		file = buf.String()
	}
	f := filepath.Join(dir, "vince.conf")
	err = ioutil.WriteFile(f, []byte(file), 0600)
	if err != nil {
		return nil, nil, err
	}
	err = format(f, formatOption{write: true})
	if err != nil {
		return nil, nil, err
	}
	return &vinceConfiguration{dir: dir, confFile: f, defaultPort: 8000}, func() { os.RemoveAll(dir) }, nil
}

func expandPort(num int) (int, error) {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", num))
	if err != nil {
		return 0, err
	}
	l.Close()
	return num, nil
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"join": filepath.Join,
		"port": expandPort,
	}
}

var httpGlobalTpl = template.Must(template.New("http").Funcs(funcMap()).Parse(`
root {{.}};
client_body_temp_path {{join . "client_body_temp"}};
fastcgi_temp_path {{join . "fastcgi_temp"}};
proxy_temp_path {{join . "proxy_temp"}};
uwsgi_temp_path {{join . "uwsgi_temp"}};
scgi_temp_path {{join . "scgi_temp"}};
`))

func testHTTPpGlobals(dir string) (string, error) {
	var buf bytes.Buffer
	err := httpGlobalTpl.Execute(&buf, dir)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func TestHTTPGlobals(t *testing.T) {
	s, err := testHTTPpGlobals("/test")
	if err != nil {
		t.Fatal(err)
	}
	expectFile(t, "fixture/test_http_globals", []byte(s))
}

func expectFile(t *testing.T, file string, expect []byte) {
	t.Helper()
	b, err := ioutil.ReadFile(file)
	if err != nil {
		t.Error(err)
		return
	}
	if !bytes.Equal(b, expect) {
		t.Errorf("check  file content: \n expected %s \ngot \n%s", string(expect), string(b))
	}
}
