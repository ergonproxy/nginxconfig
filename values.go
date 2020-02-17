package main

import (
	"strings"
	"text/template"
	"time"

	"github.com/ergongate/vince/buffers"
)

type stringValue struct {
	set   bool
	value string
}

func (s *stringValue) store(v string) {
	s.value = v
	s.set = true
}

func (s stringValue) merge(other stringValue) stringValue {
	if other.set {
		s.value = other.value
	}
	return s
}

type intValue struct {
	set   bool
	value int64
}

func (s *intValue) store(v int64) {
	s.value = v
	s.set = true
}

func (s intValue) merge(other intValue) intValue {
	if other.set {
		s.value = other.value
	}
	return s
}

type boolValue struct {
	set   bool
	value bool
}

func (s *boolValue) store(v bool) {
	s.value = v
	s.set = true
}

func (s boolValue) merge(other boolValue) boolValue {
	if other.set {
		s.value = other.value
	}
	return s
}

type stringSliceValue struct {
	set   bool
	value []string
}

func (s *stringSliceValue) store(v ...string) {
	s.value = append(s.value, v...)
	s.set = true
}

func (s stringSliceValue) merge(other stringSliceValue) stringSliceValue {
	if other.set {
		s.value = other.value
	}
	return s
}

type durationValue struct {
	set   bool
	value time.Duration
}

func (s *durationValue) store(v time.Duration) {
	s.value = v
	s.set = true
}

func (s durationValue) merge(other durationValue) durationValue {
	if other.set {
		s.value = other.value
	}
	return s
}

type interfaceValue struct {
	set   bool
	value interface{}
}

func (s *interfaceValue) store(v interfaceValue) {
	s.value = v
	s.set = true
}

func (s interfaceValue) merge(other interfaceValue) interfaceValue {
	if other.set {
		s.value = other.value
	}
	return s
}

type stringTemplateValue struct {
	value string
	set   bool
	tpl   *template.Template
}

func (s *stringTemplateValue) store(v string) {
	if !strings.Contains(v, "$") {
		s.set = true
		s.value = v
		return
	}
	x := variableRegexp.ReplaceAllFunc(src, func(name []byte) []byte {
		o := make([]byte, len(name)+5)
		o = append(o, []byte("{{.$"))
		o = append(o, name...)
		o = append(o, []byte("}}"))
		return o
	})
	s.tpl = template.Must(template.New("").Parse(string(x)))
	s.set = true
}

func (s *stringTemplateValue) Value(ctx interface{}) string {
	buf := buffers.GetBytes()
	defer buffers.PutBytes(buf)
	if s.tpl == nil {
		return s.value
	}
	if err := s.tpl.Execute(buf, ctx); err != nil {
		panic(err)
	}
	return buf.String()
}
