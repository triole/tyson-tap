package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseFilter(t *testing.T) {
	validateParseFilter(
		"front_matter.title==this+is+a+title",
		tIDXParamsFilter{
			Prefix: "front_matter.title", Operator: "==", Suffix: []string{"this+is+a+title"},
		}, t,
	)
	validateParseFilter(
		"front_matter.title!=not+the+title",
		tIDXParamsFilter{
			Prefix: "front_matter.title", Operator: "!=", Suffix: []string{"not+the+title"},
		}, t,
	)
	validateParseFilter(
		"front_matter.tags=*title2",
		tIDXParamsFilter{
			Prefix: "front_matter.tags", Operator: "=*", Suffix: []string{"title2"},
		}, t,
	)
	validateParseFilter(
		"front_matter.tags!=*title",
		tIDXParamsFilter{
			Prefix: "front_matter.tags", Operator: "!=*", Suffix: []string{"title"},
		}, t,
	)
	validateParseFilter(
		"front_matter.tags!==tag1,tag2",
		tIDXParamsFilter{
			Prefix: "front_matter.tags", Operator: "!==", Suffix: []string{"tag1", "tag2"},
		}, t,
	)
	validateParseFilter(
		"front_matter.title=~title[0-9]{1,2}",
		tIDXParamsFilter{
			Prefix: "front_matter.title", Operator: "=~", Suffix: []string{"title[0-9]{1,2}"},
		}, t,
	)
}

func validateParseFilter(filter string, exp tIDXParamsFilter, t *testing.T) {
	res := parseFilterString(filter)
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
	validateServeContent("/all.json", loadJSONArr("validate/server/all.json"), t)
}

func validateServeContent(url string, exp []string, t *testing.T) {
	CLI.LogLevel = "trace"
	CLI.Threads = 16
	conf = readConfig(fromTestFolder("conf.yaml"))
	svr := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			serveContent(w, r)
		}))
	defer svr.Close()

	c := NewClient(svr.URL)
	res, err := http.Get(c.url + url)
	if err != nil {
		t.Errorf("test serve content, request failed: %s, %s", url, err)
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("test serve content failed, can not read body: %+v", err)
	} else {
		var ji tJoinerIndex
		err = json.Unmarshal([]byte(bodyBytes), &ji)
		if err != nil {
			t.Errorf(
				"test joiner index failed, can not unmarshal server response: %+v", err,
			)
		}
		validateJoinerIndex(ji, exp, t)
	}
}

func validateJoinerIndex(ji tJoinerIndex, exp []string, t *testing.T) {
	if len(ji) != len(exp) {
		t.Errorf(
			"validate joiner index failed, lengths do not match: exp: %+v, res: %+v",
			len(exp), len(ji),
		)
	} else {
		for i := 1; i < len(ji)-1; i++ {
			if ji[i].Path != exp[i] {
				t.Errorf(
					"validate joiner index failed: exp: %+v, res: %+v", exp[i], ji[i],
				)
			}
		}
	}
}

type Client struct {
	url string
}

func NewClient(url string) Client {
	return Client{url}
}
