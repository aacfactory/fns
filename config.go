/*
 * Copyright 2021 Wang Min Xiang
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package fns

import (
	"bytes"
	"fmt"
	"github.com/goccy/go-yaml"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// +-------------------------------------------------------------------------------------------------------------------+

type ApplicationConfig struct {
	Name      string          `json:"name,omitempty"`
	Http      HttpConfig      `json:"http,omitempty"`
	Work      WorkConfig      `json:"work,omitempty"`
	Log       LogConfig       `json:"log,omitempty"`
	Discovery DiscoveryConfig `json:"discovery,omitempty"`
}

type HttpConfig struct {
	Host                     string    `json:"host,omitempty"`
	Port                     int       `json:"port,omitempty"`
	PublicHost               string    `json:"publicHost,omitempty"`
	PublicPort               int       `json:"publicPort,omitempty"`
	MaxRequestBodySizeKB     int       `json:"maxRequestBodySizeKB,omitempty"`
	MaxRequestFileSizeMB     int       `json:"maxRequestFileSizeMB,omitempty"`
	MaxConnectionsPerIP      int       `json:"maxConnectionsPerIp,omitempty"`
	MaxRequestsPerConnection int       `json:"maxRequestsPerConnection,omitempty"`
	RequestTimeoutSeconds    int       `json:"requestTimeoutSeconds,omitempty"`
	WhiteCIDR                []string  `json:"whiteCIDR,omitempty"`
	SSL                      ServerTLS `json:"ssl,omitempty"`
}

type WorkConfig struct {
	Concurrency       int `json:"concurrency,omitempty"`
	MaxIdleTimeSecond int `json:"maxIdleTimeSecond,omitempty"`
	// Aggressively reduces memory usage at the cost of higher CPU usage
	// if set to true.
	//
	// Try enabling this option only if the server consumes too much memory
	// serving mostly idle keep-alive connections. This may reduce memory
	// usage by more than 50%.
	//
	// Aggressive memory usage reduction is disabled by default.
	ReduceMemoryUsage bool `json:"reduceMemoryUsage,omitempty"`
}

// +-------------------------------------------------------------------------------------------------------------------+

type Config interface {
	As(v interface{}) (err error)
	Get(path string, v interface{}) (err error)
	Raw() (raw []byte)
}

type JsonConfig struct {
	raw []byte
}

func (config *JsonConfig) As(v interface{}) (err error) {
	decodeErr := JsonAPI().Unmarshal(config.raw, v)
	if decodeErr != nil {
		err = fmt.Errorf("fns decode config as %v failed, %s, %v", v, string(config.raw), decodeErr)
	}
	return
}

func (config *JsonConfig) Get(path string, v interface{}) (err error) {
	result := gjson.GetBytes(config.raw, path)
	if !result.Exists() {
		err = fmt.Errorf("fns config get %s failed, not exists", path)
		return
	}
	decodeErr := JsonAPI().UnmarshalFromString(result.Raw, v)
	if decodeErr != nil {
		err = fmt.Errorf("fns config get %s failed, %v", path, decodeErr)
	}
	return
}

func (config *JsonConfig) Raw() (raw []byte) {
	raw = config.raw
	return
}

type YamlConfig struct {
	raw []byte
}

func (config *YamlConfig) As(v interface{}) (err error) {
	decodeErr := yaml.Unmarshal(config.raw, v)
	if decodeErr != nil {
		err = fmt.Errorf("fns decode config as %v failed, %s, %v", v, string(config.raw), decodeErr)
	}
	return
}

func (config *YamlConfig) Get(path string, v interface{}) (err error) {
	yamlPath, pathErr := yaml.PathString(path)
	if pathErr != nil {
		err = fmt.Errorf("fns config get %s failed, bad path, %v", path, pathErr)
	}
	readErr := yamlPath.Read(bytes.NewReader(config.raw), v)
	if readErr != nil {
		err = fmt.Errorf("fns config get %s failed, %v", path, readErr)
	}
	return
}

func (config *YamlConfig) Raw() (raw []byte) {
	raw = config.raw
	return
}

// ---------------------------------------------------------------------------------------------------------------------

type ConfigStore interface {
	Read() (root []byte, subs map[string][]byte, err error)
}

func NewConfigFileStore(configPath string) ConfigStore {
	return &ConfigFileStore{
		configPath: configPath,
	}
}

type ConfigFileStore struct {
	configPath string
}

func (store *ConfigFileStore) Read() (root []byte, subs map[string][]byte, err error) {
	file, openErr := os.Open(store.configPath)
	if openErr != nil {
		err = fmt.Errorf("fns config file store open %s failed, %v", store.configPath, openErr)
		return
	}
	fileStat, statErr := file.Stat()
	if statErr != nil {
		err = fmt.Errorf("fns config file store get %s file info failed, %v", store.configPath, statErr)
		return
	}
	if !fileStat.IsDir() {
		_ = file.Close()
		fileContent, readErr := ioutil.ReadFile(store.configPath)
		if readErr != nil {
			err = fmt.Errorf("fns config file store read %s failed, %v", store.configPath, readErr)
			return
		}
		root = fileContent
		return
	}
	subs = make(map[string][]byte)
	dirErr := filepath.Walk(store.configPath, func(path string, info fs.FileInfo, cause error) (err error) {
		if info.IsDir() {
			return
		}
		if strings.Index(path, "fns") != 0 {
			return
		}
		fileContent, readErr := ioutil.ReadFile(path)
		if readErr != nil {
			err = fmt.Errorf("read %s failed, %v", path, readErr)
			return
		}
		key := store.configPath[strings.LastIndexByte(path, '/')+1 : strings.LastIndexByte(path, '.')]
		if !strings.Contains(key, "-") {
			root = fileContent
			return
		}
		key = strings.Split(key, "-")[1]
		subs[strings.ToUpper(strings.TrimSpace(key))] = fileContent
		return
	})
	if dirErr != nil {
		err = fmt.Errorf("fns config file store read %s dir failed, %v", store.configPath, dirErr)
		return
	}
	return
}

var defaultConfigRetrieverOption = ConfigRetrieverOption{
	Active: "",
	Format: "YAML",
	Store:  NewConfigFileStore("./config/fns.yaml"),
}

type ConfigRetrieverOption struct {
	Active string
	Format string
	Store  ConfigStore
}

func NewConfigRetriever(option ConfigRetrieverOption) (retriever *ConfigRetriever, err error) {
	format := strings.ToUpper(strings.TrimSpace(option.Format))
	if format == "" || !(format == "JSON" || format == "YAML") {
		err = fmt.Errorf("fns create config retriever failed, format is not support")
		return
	}
	store := option.Store
	if store == nil {
		err = fmt.Errorf("fns create config retriever failed, store is nil")
		return
	}
	retriever = &ConfigRetriever{
		active: strings.ToUpper(strings.TrimSpace(option.Active)),
		format: format,
		store:  store,
	}
	return
}

type ConfigRetriever struct {
	active string
	format string
	store  ConfigStore
}

func (retriever ConfigRetriever) Get() (config Config, err error) {
	root, subs, readErr := retriever.store.Read()
	if readErr != nil {
		err = fmt.Errorf("fns config retriever get failed, %v", readErr)
		return
	}
	if root == nil || len(root) == 0 {
		err = fmt.Errorf("fns config retriever get failed, not found")
		return
	}
	if retriever.active == "" {
		if retriever.format == "JSON" {
			if !gjson.ValidBytes(root) {
				err = fmt.Errorf("fns config retriever get failed, bad json content")
				return
			}
			config = &JsonConfig{
				raw: root,
			}
		} else if retriever.format == "YAML" {
			_, validErr := yaml.YAMLToJSON(root)
			if validErr != nil {
				err = fmt.Errorf("fns config retriever get failed, bad yaml content")
				return
			}
			config = &YamlConfig{
				raw: root,
			}
		} else {
			err = fmt.Errorf("fns config retriever get failed, format is unsupported")
			return
		}
	}

	if subs == nil || len(subs) == 0 {
		err = fmt.Errorf("fns config retriever get failed, ative(%s) is not found", retriever.active)
		return
	}

	sub, hasSub := subs[retriever.active]
	if !hasSub {
		err = fmt.Errorf("fns config retriever get failed, ative(%s) is not found", retriever.active)
		return
	}

	mergedConfig, mergeErr := retriever.merge(retriever.format, root, sub)
	if mergeErr != nil {
		err = fmt.Errorf("fns config retriever get failed, merge ative failed %v", mergeErr)
		return
	}

	config = mergedConfig
	return
}

func (retriever ConfigRetriever) merge(format string, root []byte, sub []byte) (config Config, err error) {
	if format == "JSON" {
		config, err = retriever.mergeJson(root, sub)
	} else if format == "YAML" {
		config, err = retriever.mergeYaml(root, sub)
	} else {
		err = fmt.Errorf("format is unsupported")
		return
	}
	return
}

func (retriever ConfigRetriever) mergeJson(root []byte, sub []byte) (config Config, err error) {
	if !gjson.ValidBytes(root) {
		err = fmt.Errorf("merge failed, bad json content")
		return
	}
	if !gjson.ValidBytes(sub) {
		err = fmt.Errorf("merge failed, bad json content")
		return
	}
	subResult := gjson.ParseBytes(sub)
	subResult.ForEach(func(key gjson.Result, value gjson.Result) bool {
		root0, setErr := sjson.SetRawBytes(root, key.String(), []byte(value.Raw))
		if setErr != nil {
			return false
		}
		root = root0
		return true
	})

	return
}

func (retriever ConfigRetriever) mergeYaml(root []byte, sub []byte) (config Config, err error) {
	rootJson, rootToJsonErr := yaml.YAMLToJSON(root)
	if rootToJsonErr != nil {
		err = fmt.Errorf("merge failed, content format is not supported, %v", rootToJsonErr)
		return
	}
	subJson, subToJsonErr := yaml.YAMLToJSON(sub)
	if subToJsonErr != nil {
		err = fmt.Errorf("merge failed, content format is not supported, %v", subToJsonErr)
		return
	}
	jsonConfig, mergeJsonErr := retriever.mergeJson(rootJson, subJson)
	if mergeJsonErr != nil {
		err = mergeJsonErr
		return
	}
	yamlContent, toYamlErr := yaml.JSONToYAML(jsonConfig.Raw())
	if toYamlErr != nil {
		err = fmt.Errorf("merge failed, transfer failed, %v", toYamlErr)
		return
	}
	config = &YamlConfig{
		raw: yamlContent,
	}
	return
}
