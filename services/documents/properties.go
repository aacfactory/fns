package documents

import (
	"sort"
	"strings"
)

type Property struct {
	Name    string
	Element Element
}

type Properties []Property

func (pp Properties) Len() int {
	return len(pp)
}

func (pp Properties) Less(i, j int) bool {
	return strings.Compare(pp[i].Name, pp[j].Name) < 0
}

func (pp Properties) Swap(i, j int) {
	pp[i], pp[j] = pp[j], pp[i]
}

func (pp Properties) Add(name string, element Element) Properties {
	n := append(pp, Property{
		Name:    name,
		Element: element,
	})
	sort.Sort(n)
	return n
}

func (pp Properties) Get(name string) (p Property, has bool) {
	for _, property := range pp {
		if property.Name == name {
			p = property
			has = true
			return
		}
	}
	return
}
