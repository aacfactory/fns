package sources

import (
	"bufio"
	"bytes"
	"github.com/aacfactory/errors"
	"io"
	"strings"
)

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

func (annotations *Annotations) set(name string, params string) {
	params, _ = strings.CutSuffix(params, "\n")
	params = strings.ReplaceAll(params, "'>>>'", ">>>")
	params = strings.ReplaceAll(params, "'<<<'", "<<<")
	ss := *annotations
	ss = append(ss, Annotation{
		Name:   name,
		Params: []string{params},
	})
}

func ParseAnnotations(s string) (annotations Annotations, err error) {
	annotations = make(Annotations, 0, 1)
	if s == "" || !strings.Contains(s, "@") {
		return
	}
	currentKey := ""
	currentBody := bytes.NewBuffer(make([]byte, 0, 1))
	blockReading := false
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
		if line == nil {
			continue
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if line[0] == '@' {
			if blockReading {
				currentBody.Write(line)
				if len(line) > 0 {
					currentBody.WriteByte('\n')
				}
				continue
			}
			if len(line) == 1 {
				continue
			}
			if currentKey != "" {
				annotations.set(currentKey, currentBody.String())
				currentKey = ""
				currentBody.Reset()
			}
			idx := bytes.IndexByte(line, ' ')
			if idx < 0 {
				currentKey = string(line[1:])
				continue
			}
			currentKey = string(line[1:idx])
			line = bytes.TrimSpace(line[idx:])
		}
		if len(line) == 0 {
			continue
		}
		if blockReading {
			remains, hasBlockEnd := bytes.CutSuffix(line, []byte{'<', '<', '<'})
			currentBody.Write(remains)
			if hasBlockEnd {
				annotations.set(currentKey, currentBody.String())
				currentKey = ""
				currentBody.Reset()

				blockReading = false
			} else {
				if len(remains) > 0 {
					currentBody.WriteByte('\n')
				}
			}
			continue
		}
		line, blockReading = bytes.CutPrefix(line, []byte{'>', '>', '>'})
		if blockReading && currentKey != "" {
			remains, hasBlockEnd := bytes.CutSuffix(line, []byte{'<', '<', '<'})
			currentBody.Write(remains)
			if hasBlockEnd {
				annotations.set(currentKey, currentBody.String())
				currentKey = ""
				currentBody.Reset()

				blockReading = false
			} else {
				if len(remains) > 0 {
					currentBody.WriteByte('\n')
				}
			}
			continue
		} else if currentKey != "" {
			currentBody.Write(line)

			annotations.set(currentKey, currentBody.String())
			currentKey = ""
			currentBody.Reset()
		}

	}
	if currentKey != "" {
		annotations.set(currentKey, currentBody.String())
		currentKey = ""
		currentBody.Reset()
	}
	return
}
