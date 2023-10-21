package configs

import (
	"fmt"
	"github.com/aacfactory/configures"
	"github.com/aacfactory/errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	activeSystemEnvKey = "FNS-ACTIVE"
)

func DefaultConfigRetrieverOption() (option configures.RetrieverOption) {
	path, pathErr := filepath.Abs("./configs")
	if pathErr != nil {
		panic(fmt.Errorf("%+v", errors.Warning("fns: create default config retriever failed, cant not get absolute representation of './configs'").WithCause(pathErr)))
		return
	}
	active, _ := os.LookupEnv(activeSystemEnvKey)
	active = strings.TrimSpace(active)
	store := configures.NewFileStore(path, "fns", '-')
	option = configures.RetrieverOption{
		Active: active,
		Format: "YAML",
		Store:  store,
	}
	return
}
