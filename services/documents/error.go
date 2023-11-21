package documents

import (
	"bufio"
	"bytes"
	"io"
	"sort"
	"strings"
)

type ErrorDescription struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ErrorDescriptions []ErrorDescription

func (pp ErrorDescriptions) Len() int {
	return len(pp)
}

func (pp ErrorDescriptions) Less(i, j int) bool {
	return strings.Compare(pp[i].Name, pp[j].Name) < 0
}

func (pp ErrorDescriptions) Swap(i, j int) {
	pp[i], pp[j] = pp[j], pp[i]
}

func (pp ErrorDescriptions) Add(name string, value string) ErrorDescriptions {
	n := append(pp, ErrorDescription{
		Name:  name,
		Value: value,
	})
	sort.Sort(n)
	return n
}

func (pp ErrorDescriptions) Get(name string) (v string, has bool) {
	for _, description := range pp {
		if description.Name == name {
			v = description.Value
			has = true
			return
		}
	}
	return
}

func NewError(name string) Error {
	return Error{
		Name:         name,
		Descriptions: make(ErrorDescriptions, 0, 1),
	}
}

type Error struct {
	Name         string            `json:"name"`
	Descriptions ErrorDescriptions `json:"descriptions"`
}

func (err Error) AddNamedDescription(name string, value string) Error {
	err.Descriptions = err.Descriptions.Add(name, value)
	return err
}

func NewErrors(src string) Errors {
	errs := make(Errors, 0, 1)
	if len(src) == 0 {
		return errs
	}
	reader := bufio.NewReader(bytes.NewReader([]byte(src)))
	pos := -1
	for {
		line, _, readErr := reader.ReadLine()
		if readErr == io.EOF {
			break
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		idx := bytes.IndexByte(line, ':')
		if idx < 0 {
			errs = errs.Add(NewError(string(line)))
			pos++
			continue
		}
		name := bytes.TrimSpace(line[0:idx])
		var value []byte
		if len(line) > idx+1 {
			value = bytes.TrimSpace(line[idx+1:])
		}
		errs[pos] = errs[pos].AddNamedDescription(string(name), string(value))
	}
	return errs
}

// Errors
// use @errors
// @errors >>>
//
//	name1
//	zh: chinese
//	en: english
//	name2
//	zh: chinese
//	en: english
//
// <<<
type Errors []Error

func (pp Errors) Len() int {
	return len(pp)
}

func (pp Errors) Less(i, j int) bool {
	return strings.Compare(pp[i].Name, pp[j].Name) < 0
}

func (pp Errors) Swap(i, j int) {
	pp[i], pp[j] = pp[j], pp[i]
}

func (pp Errors) Add(e Error) Errors {
	n := append(pp, e)
	sort.Sort(n)
	return n
}
