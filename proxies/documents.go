package proxies

import "github.com/aacfactory/fns/services/documents"

type Documents []documents.Documents

func (d Documents) Len() int {
	return len(d)
}

func (d Documents) Less(i, j int) bool {
	return d[i].Version.LessThan(d[j].Version)
}

func (d Documents) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}
