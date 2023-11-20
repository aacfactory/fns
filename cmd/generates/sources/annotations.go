package sources

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/aacfactory/errors"
	"io"
	"strings"
)

func NewAnnotation(name string, params ...string) Annotation {
	var pp []string
	if len(params) > 0 {
		pp = params
	}
	return Annotation{
		Name:   name,
		Params: pp,
	}
}

type Annotation struct {
	Name   string
	Params []string
}

type Annotations []Annotation

func (annotations *Annotations) Get(name string) (annotation Annotation, has bool) {
	ss := *annotations
	for _, target := range ss {
		if target.Name == name {
			annotation = target
			has = true
			return
		}
	}
	return
}

func (annotations *Annotations) Add(name string, param string) {
	ss := *annotations
	for i, s := range ss {
		if s.Name == name {
			if param == "" {
				return
			}
			s.Params = append(s.Params, param)
			ss[i] = s
			*annotations = ss
			return
		}
	}
	params := make([]string, 0, 1)
	if param != "" {
		params = append(params, param)
	}
	ss = append(ss, Annotation{
		Name:   name,
		Params: params,
	})
	*annotations = ss
}

func (annotations *Annotations) Set(name string, param string) {
	if param != "" {
		param, _ = strings.CutSuffix(param, "\n")
		param = strings.ReplaceAll(param, "'>>>'", ">>>")
		param = strings.ReplaceAll(param, "'<<<'", "<<<")
		param = strings.TrimSpace(param)
	}
	ss := *annotations
	for i, s := range ss {
		if s.Name == name {
			params := make([]string, 0, 1)
			if param != "" {
				params = append(params, param)
			}
			s.Params = params
			ss[i] = s
			*annotations = ss
			return
		}
	}
	params := make([]string, 0, 1)
	if param != "" {
		params = append(params, param)
	}
	ss = append(ss, Annotation{
		Name:   name,
		Params: params,
	})
	*annotations = ss
}

func ParseAnnotations(s string) (annotations Annotations, err error) {
	annotations = make(Annotations, 0, 1)
	if s == "" || !strings.Contains(s, "@") {
		return
	}
	reader := bufio.NewReader(bytes.NewReader([]byte(s)))
	for {
		line, _, readErr := reader.ReadLine()
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			err = errors.Warning("sources: parse annotations failed").WithCause(readErr).WithMeta("source", s)
			return
		}
		if len(line) == 0 {
			continue
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if line[0] != '@' {
			continue
		}
		name := ""

		content := line[1:]
		paramsIdx := bytes.IndexByte(content, ' ')
		if paramsIdx < 1 {
			// no params
			name = string(content)
			_, has := annotations.Get(name)
			if has {
				err = errors.Warning("sources: parse annotations failed").WithCause(fmt.Errorf("@%s is duplicated", name)).WithMeta("source", s)
				return
			}
			annotations.Set(name, "")
			continue
		} else {
			name = strings.TrimSpace(string(content[0:paramsIdx]))
			_, has := annotations.Get(name)
			if has {
				err = errors.Warning("sources: parse annotations failed").WithCause(fmt.Errorf("@%s is duplicated", name)).WithMeta("source", s)
				return
			}
			content = bytes.TrimSpace(content[paramsIdx:])
			if bytes.Index(content, []byte(">>>")) == 0 {
				// block
				block := bytes.NewBuffer(make([]byte, 0, 1))
				block.Write([]byte{'\n'})
				content = bytes.TrimSpace(content[3:])
				if endIdx := bytes.Index(content, []byte("<<<")); endIdx > -1 {
					content = bytes.TrimSpace(content[0:endIdx])
					block.Write(content)
					continue
				}
				if len(content) > 0 {
					block.Write(content)
				}
				for {
					line, _, readErr = reader.ReadLine()
					if readErr != nil {
						err = errors.Warning("sources: parse annotations failed").WithCause(fmt.Errorf("@%s is incompleted", name)).WithMeta("source", s)
						return
					}
					content = bytes.TrimSpace(line)
					if len(content) == 0 {
						block.Write([]byte{'\n'})
						continue
					}
					if content[0] == '@' {
						err = errors.Warning("sources: parse annotations failed").WithCause(fmt.Errorf("@%s is incompleted", name)).WithMeta("source", s)
						return
					}
					if bytes.Index(content, []byte("<<<")) > -1 {
						content, _ = bytes.CutSuffix(content, []byte("<<<"))
						content = bytes.TrimSpace(content)
						block.Write([]byte{'\n'})
						block.Write(content)
						break
					}
					block.Write([]byte{'\n'})
					block.Write(content)
				}
				annotations.Set(name, block.String()[1:])
			} else {
				params := strings.Split(string(content), " ")
				for _, param := range params {
					param = strings.TrimSpace(param)
					annotations.Add(name, param)
				}
			}
		}
	}
	return
}
