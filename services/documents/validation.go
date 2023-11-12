package documents

import (
	"sort"
	"strings"
)

type I18n struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type I18ns []I18n

func (pp I18ns) Len() int {
	return len(pp)
}

func (pp I18ns) Less(i, j int) bool {
	return strings.Compare(pp[i].Name, pp[j].Name) < 0
}

func (pp I18ns) Swap(i, j int) {
	pp[i], pp[j] = pp[j], pp[i]
}

func (pp I18ns) Add(name string, value string) I18ns {
	n := append(pp, I18n{
		Name:  name,
		Value: value,
	})
	sort.Sort(n)
	return n
}

func (pp I18ns) Get(name string) (p string, has bool) {
	for _, i18n := range pp {
		if i18n.Name == name {
			p = i18n.Value
			has = true
			return
		}
	}
	return
}

func NewValidation(name string) Validation {
	return Validation{
		Enable: true,
		Name:   name,
		I18ns:  make(I18ns, 0, 1),
	}
}

type Validation struct {
	Enable bool   `json:"enable,omitempty"`
	Name   string `json:"name,omitempty"`
	I18ns  I18ns  `json:"i18ns,omitempty"`
}

func (validation Validation) AddI18n(name string, value string) Validation {
	validation.I18ns = validation.I18ns.Add(name, value)
	return validation
}
