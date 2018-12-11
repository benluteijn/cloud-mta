// Package mta provides a convenient way of exploring the structure of `mta.yaml` file objects
// such as retrieving a list of resources required by a specific module.
package mta

import (
	"encoding/json"
	"fmt"
	yaml2 "github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

// GetModules returns a list of MTA modules.
func (mta *MTA) GetModules() []*Module {
	return mta.Modules
}

// GetResources returns list of MTA resources.
func (mta *MTA) GetResources() []*Resource {
	return mta.Resources
}

// GetModuleByName returns a specific module by name.
func (mta *MTA) GetModuleByName(name string) (*Module, error) {
	for _, m := range mta.Modules {
		if m.Name == name {
			return m, nil
		}
	}
	return nil, fmt.Errorf("module %s , not found ", name)
}

// GetModuleByName returns a specific module by name from extension object
func (ext *EXT) GetModuleByName(name string) (*ModuleExt, error) {
	for _, m := range ext.Modules {
		if m.Name == name {
			return m, nil
		}
	}
	return nil, fmt.Errorf("module %s , not found ", name)
}

// GetResourceByName returns a specific resource by name.
func (mta *MTA) GetResourceByName(name string) (*Resource, error) {
	for _, r := range mta.Resources {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, fmt.Errorf("module %s , not found ", name)
}

// Unmarshal returns a reference to the MTA object from a byte array.
func Unmarshal(content []byte) (*MTA, error) {
	m := &MTA{}
	// Unmarshal MTA file
	err := yaml.Unmarshal([]byte(content), &m)
	if err != nil {
		err = errors.Wrap(err, "Error parsing the MTA")
	}
	return m, err
}

// UnmarshalExt returns a reference to the EXT object from a byte array.
func UnmarshalExt(content []byte) (*EXT, error) {
	m := &EXT{}
	// Unmarshal MTA file
	err := yaml.Unmarshal([]byte(content), &m)
	if err != nil {
		err = errors.Wrap(err, "Error parsing the MTA")
	}
	return m, err
}

// Merge merges mta object with mta extension object
// extension properties complement and overwrite mta properties
func Merge(mta *MTA, mtaExt *EXT) {
	for _, module := range mta.Modules {
		extModule, err := mtaExt.GetModuleByName(module.Name)
		if err == nil {
			extendMap(&module.Properties, &extModule.Properties)
			extendMap(&module.Parameters, &extModule.Parameters)
			extendMap(&module.BuildParams, &extModule.BuildParams)
		}
	}
}

// extendMap extends map with elements of mta extension map
func extendMap(m *map[string]interface{}, ext *map[string]interface{}) {
	if *m == nil {
		*m = make(map[string]interface{})
	}
	if ext != nil {
		for key, value := range *ext {
			(*m)[key] = value
		}
	}
}

// YamlToJson converting YAML content to JSON content and generates JSON file
func YamlToJson(outFileName string, content []byte) ([]byte, error) {
	var jsonRawEscaped json.RawMessage   // json raw with escaped unicode chars
	var jsonRawUnescaped json.RawMessage // json raw with unescaped unicode chars
	//Converting YAML content to JSON content
	content, err := yaml2.YAMLToJSON(content)
	if err != nil {
		err = errors.Wrap(err, "Error converting YAML to JSON")
	}
	//Avoid escaping characters
	jsonRawEscaped = []byte(string(content))
	jsonRawUnescaped, err = _UnescapeUnicodeCharactersInJSON(jsonRawEscaped)
	if err != nil {
		err = errors.Wrap(err, "Error unescape unicode characters in JSON")
	}
	os.Create(outFileName)
	err = ioutil.WriteFile(outFileName, jsonRawUnescaped, 0644)
	if err != nil {
		err = errors.Wrap(err, "Error Writing to JSON file")
	}
	return jsonRawUnescaped, err
}

//_UnescapeUnicodeCharactersInJSON avoids escaping characters in JSON format
func _UnescapeUnicodeCharactersInJSON(_jsonRaw json.RawMessage) (json.RawMessage, error) {
	str, err := strconv.Unquote(strings.Replace(strconv.Quote(string(_jsonRaw)), `\\u`, `\u`, -1))
	if err != nil {
		return nil, err
	}
	return []byte(str), nil
}