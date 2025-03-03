package indexer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestParseFilter(t *testing.T) {
	tc := InitTests(false)

	tc.validateParseFilter(
		"front_matter.title==this+is+a+title",
		FilterParams{
			Prefix: "front_matter.title", Operator: "==", Suffix: []string{"this+is+a+title"},
		}, t,
	)
	tc.validateParseFilter(
		"front_matter.title!=not+the+title",
		FilterParams{
			Prefix: "front_matter.title", Operator: "!=", Suffix: []string{"not+the+title"},
		}, t,
	)
	tc.validateParseFilter(
		"front_matter.tags=*title2",
		FilterParams{
			Prefix: "front_matter.tags", Operator: "=*", Suffix: []string{"title2"},
		}, t,
	)
	tc.validateParseFilter(
		"front_matter.tags!=*title",
		FilterParams{
			Prefix: "front_matter.tags", Operator: "!=*", Suffix: []string{"title"},
		}, t,
	)
	tc.validateParseFilter(
		"front_matter.tags!==tag1,tag2",
		FilterParams{
			Prefix: "front_matter.tags", Operator: "!==", Suffix: []string{"tag1", "tag2"},
		}, t,
	)
	tc.validateParseFilter(
		"front_matter.title=~title[0-9]{1,2}",
		FilterParams{
			Prefix: "front_matter.title", Operator: "=~", Suffix: []string{"title[0-9]{1,2}"},
		}, t,
	)
}

func (tc testContext) validateParseFilter(filter string, exp FilterParams, t *testing.T) {
	res := tc.ind.parseFilterString(filter)
	r := true
	if res.Prefix != exp.Prefix || res.Operator != exp.Operator {
		r = false
		for i := 0; i < len(exp.Suffix); i++ {
			if res.Suffix[i] != exp.Suffix[i] {
				r = false
			}
		}
	}
	if !r {
		t.Errorf("parse filter failed, \nexp: %+v\ngot: %+v", exp, res)
	}
}

func TestServeContent(t *testing.T) {
	tc := InitTests(false)
	fol := ut.FromTestFolder("specs/server")
	specFiles, err := tc.ind.Util.Find(fol, "\\.yaml$")
	if err != nil {
		t.Errorf("can not find specs files: %q", fol)
	}

	for _, specFile := range specFiles {
		tc.validateServeContent(specFile, t)
	}
}

func (tc testContext) validateServeContent(specFile string, t *testing.T) {
	specs := tc.readSpecs(specFile)
	for _, specItf := range specs {
		spec := specItf.(map[string]interface{})
		urls := tc.ind.Util.ListInterfaceToListString(
			spec["urls"].([]interface{}),
		)
		exp := tc.ind.Util.ListInterfaceToListString(
			spec["exp"].([]interface{}),
		)
		testsrv := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				tc.ind.serveContent(w, r)
			}))
		defer testsrv.Close()

		for _, url := range urls {
			tc.ind.flushCache()
			c := newClient(testsrv.URL)
			res, err := http.Get(c.url + url)
			if err != nil {
				t.Errorf("test serve content, request failed: %s, %s", url, err)
			}
			defer res.Body.Close()

			bodyBytes, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("test serve content failed, can not read body: %+v", err)
			} else {
				var ti TapIndex
				err = json.Unmarshal([]byte(bodyBytes), &ti)
				if err != nil {
					t.Errorf(
						"test joiner index failed, can not unmarshal server response: %+v", err,
					)
				}
				tc.validateTapIndex(ti, url, exp, t)
			}
		}
	}
}

func (tc testContext) validateTapIndex(ti TapIndex, url string, exp []string, t *testing.T) {
	failed := false
	if len(ti) != len(exp) {
		t.Errorf(
			"validate tap index failed: %q, lengths do not match: exp: %+v, got: %+v",
			url, len(exp), len(ti),
		)
	} else {
		for i := 0; i < len(ti); i++ {
			if ti[i].Path != exp[i] {
				failed = true
				break
			}
		}
	}

	if failed {
		t.Errorf(
			"validate tap index failed: %q\n"+
				"exp, len: %d\n %+v,\n"+
				"got, len: %d\n%+v\n",
			url, len(exp), exp, len(ti), tc.getTapIndexPaths(ti),
		)
	}
}

type Client struct {
	url string
}

func newClient(url string) Client {
	return Client{url}
}

func BenchmarkServer(b *testing.B) {
	tc := InitTests(false)
	pos := ut.Trace()
	testsrv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			tc.ind.serveContent(w, r)
		}))
	defer testsrv.Close()
	for urlStr := range tc.ind.Conf.API {
		// NOTE: figure out how to deal with relative URLs later
		pURL, err := url.Parse(urlStr)
		if err == nil && pURL.IsAbs() {
			c := newClient(testsrv.URL)
			http.Get(c.url + pURL.String())
		}
	}
	fmt.Printf("%s took %s with b.N = %d\n", pos, b.Elapsed(), b.N)
}
