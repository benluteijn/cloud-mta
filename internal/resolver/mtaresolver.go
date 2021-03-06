package resolver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"

	"github.com/SAP/cloud-mta/internal/logs"
	"github.com/SAP/cloud-mta/mta"
)

const (
	emptyModuleNameMsg = "provide a name for the module"
	pathNotFoundMsg    = `could not find the "%s" path`
	unmarshalFailsMsg  = `could not unmarshal the "%s"`
	moduleNotFoundMsg  = `could not find the "%s" module`
	marshalFailsMag    = `could not marshal the "%s" environment variable`
	missingPrefixMsg   = `could not resolve the value for the "~{%s}" variable; missing required prefix`

	defaultEnvFileName = ".env"
)

var envGetter = os.Environ

// Resolve - resolve module's parameters
func Resolve(workspaceDir, moduleName, modulePath string, envFile string) error {
	if len(moduleName) == 0 {
		return errors.New(emptyModuleNameMsg)
	}
	yamlData, err := ioutil.ReadFile(modulePath)
	if err != nil {
		return errors.Wrapf(err, pathNotFoundMsg, modulePath)
	}
	mtaRaw, err := mta.Unmarshal(yamlData)
	if err != nil {
		return errors.Wrapf(err, unmarshalFailsMsg, modulePath)
	}

	if len(workspaceDir) == 0 {
		workspaceDir = path.Dir(modulePath)
	}

	// If environment file name is not provided - set the default file name to .env
	envFileName := defaultEnvFileName
	if len(envFile) > 0 {
		envFileName = envFile
	}

	m := NewMTAResolver(mtaRaw, workspaceDir)

	for _, module := range m.GetModules() {
		if module.Name == moduleName {
			m.ResolveProperies(module, envFileName)

			propVarMap, err := getPropertiesAsEnvVar(module)
			if err != nil {
				return err
			}
			for key, val := range propVarMap {
				fmt.Println(key + "=" + val)
			}
			return nil
		}
	}

	return errors.Errorf(moduleNotFoundMsg, moduleName)
}

func getPropertiesAsEnvVar(module *mta.Module) (map[string]string, error) {
	envVar := map[string]interface{}{}
	for key, val := range module.Properties {
		envVar[key] = val
	}

	for _, requires := range module.Requires {
		propMap := envVar
		if len(requires.Group) > 0 {
			propMap = map[string]interface{}{}
		}

		for key, val := range requires.Properties {
			propMap[key] = val
		}

		if len(requires.Group) > 0 {
			//append the array element to group
			group, ok := envVar[requires.Group]
			if ok {
				groupArray := group.([]map[string]interface{})
				envVar[requires.Group] = append(groupArray, propMap)
			} else {
				envVar[requires.Group] = []map[string]interface{}{propMap}
			}
		}
	}

	//serialize
	return serializePropertiesAsEnvVars(envVar)
}

func serializePropertiesAsEnvVars(envVar map[string]interface{}) (map[string]string, error) {
	retEnvVar := map[string]string{}
	for key, val := range envVar {
		switch v := val.(type) {
		case string:
			retEnvVar[key] = v
		default:
			bytesVal, err := json.Marshal(val)
			if err != nil {
				return nil, errors.Errorf(marshalFailsMag, key)
			}
			retEnvVar[key] = string(bytesVal)
		}
	}

	return retEnvVar, nil
}

// MTAResolver is used to resolve MTA properties' variables
type MTAResolver struct {
	mta.MTA
	WorkingDir string
	context    *ResolveContext
}

const resourceType = 1
const moduleType = 2

const variablePrefix = "~"
const placeholderPrefix = "$"

type mtaSource struct {
	Name       string
	Parameters map[string]interface{} `yaml:"parameters,omitempty"`
	Properties map[string]interface{} `yaml:"properties,omitempty"`
	Type       int
	Module     *mta.Module
	Resource   *mta.Resource
}

// NewMTAResolver is a factory function for MTAResolver
func NewMTAResolver(m *mta.MTA, workspaceDir string) *MTAResolver {
	resolver := &MTAResolver{*m, workspaceDir, &ResolveContext{
		global:    map[string]string{},
		modules:   map[string]map[string]string{},
		resources: map[string]map[string]string{},
	}}

	for _, module := range m.Modules {
		resolver.context.modules[module.Name] = map[string]string{}
	}

	for _, resource := range m.Resources {
		resolver.context.resources[resource.Name] = map[string]string{}
	}
	return resolver
}

// ResolveProperies is the main function to trigger the resolution
func (m *MTAResolver) ResolveProperies(module *mta.Module, envFileName string) {

	if m.Parameters == nil {
		m.Parameters = map[string]interface{}{}
	}

	//add env variables
	for _, val := range envGetter() {
		pos := strings.Index(val, "=")
		if pos > 0 {
			key := strings.Trim(val[:pos], " ")
			value := strings.Trim(val[pos+1:], " ")
			m.addValueToContext(key, value)
		}
	}

	//add .env file in module's path to the module context
	if len(module.Path) > 0 {
		envFile := path.Join(m.WorkingDir, module.Path, envFileName)
		envMap, err := godotenv.Read(envFile)
		if err == nil {
			for key, value := range envMap {
				m.addValueToContext(key, value)
			}
		}
	}
	m.addServiceNames(module)

	//top level properties
	for key, value := range module.Properties {
		//no expected variables
		propValue := m.resolve(module, nil, value)
		module.Properties[key] = m.resolvePlaceholders(module, nil, nil, propValue)
	}

	//required properties:
	for _, req := range module.Requires {
		requiredSource := m.findProvider(req.Name)
		for propName, PropValue := range req.Properties {
			resolvedValue := m.resolve(module, &req, PropValue)
			//replace value with resolved value
			req.Properties[propName] = m.resolvePlaceholders(module, requiredSource, &req, resolvedValue)
		}
	}
}

func (m *MTAResolver) addValueToContext(key, value string) {
	//if the key has format of "module/key", or "resource/key" writes the value to the module's context
	slashPos := strings.Index(key, "/")
	if slashPos > 0 {
		modName := key[:slashPos]
		key = key[slashPos+1:]
		modulesContext, ok := m.context.modules[modName]
		if !ok {
			modulesContext, ok = m.context.resources[modName]
		}
		if ok {
			modulesContext[key] = value
		}
	} else {
		m.context.global[key] = value
	}

}

func (m *MTAResolver) resolve(sourceModule *mta.Module, requires *mta.Requires, valueObj interface{}) interface{} {
	switch valueObj.(type) {
	case map[interface{}]interface{}:
		v := convertToJSONSafe(valueObj)
		return m.resolve(sourceModule, requires, v)
	case map[string]interface{}:
		value := valueObj.(map[string]interface{})
		for k, v := range value {
			value[k] = m.resolve(sourceModule, requires, v)
		}
		return value
	case []interface{}:
		value := valueObj.([]interface{})
		for i, v := range value {
			value[i] = m.resolve(sourceModule, requires, v)
		}
		return value
	case string:
		return m.resolveString(sourceModule, requires, valueObj.(string))
	default:
		//if the value is not a string but a leaf, just return it
		return valueObj
	}

}

func (m *MTAResolver) resolveString(sourceModule *mta.Module, requires *mta.Requires, value string) interface{} {
	pos := 0

	pos, variableName, wholeValue := parseNextVariable(pos, value, variablePrefix)
	if pos < 0 {
		//no variables
		return value
	}
	varValue := m.getVariableValue(sourceModule, requires, variableName)

	if wholeValue {
		return varValue
	}
	for pos >= 0 {
		varValueStr, _ := convertToString(varValue)
		value = value[:pos] + varValueStr + value[pos+len(variableName)+3:]

		pos, variableName, _ = parseNextVariable(pos+len(varValueStr), value, variablePrefix)
		if pos >= 0 {
			varValue = m.getVariableValue(sourceModule, requires, variableName)
		}
	}

	return value
}

func convertToString(valueObj interface{}) (string, bool) {
	switch v := valueObj.(type) {
	case string:
		return v, false
	}
	valueBytes, err := json.Marshal(convertToJSONSafe(valueObj))
	if err != nil {
		logs.Logger.Error(err)
		return "", false
	}
	return string(valueBytes), true
}

// return start position, name of variable and if it is a whole value
func parseNextVariable(pos int, value string, prefix string) (int, string, bool) {

	endSign := "}"
	posStart := strings.Index(value[pos:], prefix+"{")
	if posStart < 0 {
		return -1, "", false
	}
	posStart += pos

	if string(value[posStart+2]) == "{" {
		endSign = "}}"
	}

	posEnd := strings.Index(value[posStart+2:], endSign)
	if posEnd < 0 {
		//bad value
		return -1, "", false
	}
	posEnd += posStart + 1 + len(endSign)
	wholeValue := posStart == 0 && posEnd == len(value)-1

	return posStart, value[posStart+2 : posEnd], wholeValue
}

func (m *MTAResolver) getVariableValue(sourceModule *mta.Module, requires *mta.Requires, variableName string) interface{} {
	var providerName string
	if requires == nil {
		slashPos := strings.Index(variableName, "/")
		if slashPos > 0 {
			providerName = variableName[:slashPos]
			variableName = variableName[slashPos+1:]
		} else {
			logs.Logger.Warnf(missingPrefixMsg, variableName)
			return "~{" + variableName + "}"
		}

	} else {
		providerName = requires.Name
	}

	source := m.findProvider(providerName)
	if source != nil {
		for propName, propValue := range source.Properties {
			if propName == variableName {

				//Do not pass module and requires, because it is a wrong scope
				//it is either global->module->requires
				//or           global->resource
				propValue = m.resolvePlaceholders(nil, source, nil, propValue)
				return convertToJSONSafe(propValue)
			}
		}
	}

	if source != nil && source.Type == resourceType && source.Resource.Type == "configuration" {
		provID, ok := source.Resource.Parameters["provider-id"]
		if ok {
			println("missing configuration ", provID.(string), "/", variableName)
		}
	}

	return "~{" + variableName + "}"
}

func (m *MTAResolver) resolvePlaceholders(sourceModule *mta.Module, source *mtaSource, requires *mta.Requires, valueObj interface{}) interface{} {
	switch valueObj.(type) {
	case map[interface{}]interface{}:
		v := convertToJSONSafe(valueObj)
		return m.resolvePlaceholders(sourceModule, source, requires, v)
	case map[string]interface{}:
		value := valueObj.(map[string]interface{})
		for k, v := range value {
			value[k] = m.resolvePlaceholders(sourceModule, source, requires, v)
		}
		return value
	case []interface{}:
		value := valueObj.([]interface{})
		for k, v := range value {
			value[k] = m.resolvePlaceholders(sourceModule, source, requires, v)
		}
		return value
	case string:
		return m.resolvePlaceholdersString(sourceModule, source, requires, valueObj.(string))
	default:
		//if the value is not a string but a leaf, just return it
		return valueObj
	}
}

func (m *MTAResolver) resolvePlaceholdersString(sourceModule *mta.Module, source *mtaSource, requires *mta.Requires, value string) interface{} {
	pos := 0
	pos, placeholderName, wholeValue := parseNextVariable(pos, value, placeholderPrefix)

	if pos < 0 {
		return value
	}
	placeholderValue := m.getParameter(sourceModule, source, requires, placeholderName)

	if wholeValue {
		return placeholderValue
	}
	for pos >= 0 {
		phValueStr, _ := convertToString(placeholderValue)
		value = value[:pos] + phValueStr + value[pos+len(placeholderName)+3:]
		pos, placeholderName, _ = parseNextVariable(pos+len(phValueStr), value, placeholderPrefix)
		if pos >= 0 {
			placeholderValue = m.getParameter(sourceModule, source, requires, placeholderName)
		}
	}

	return value
}

func (m *MTAResolver) getParameterFromSource(source *mtaSource, paramName string) string {
	if source != nil {
		paramVal := source.Parameters[paramName]
		if paramVal != nil {
			return paramVal.(string)
		}

		//defaults to context's module params:
		paramValStr, ok := m.context.modules[source.Name][paramName]
		if ok {
			return paramValStr
		}

		//defaults to context's resource params:
		paramValStr, ok = m.context.resources[source.Name][paramName]
		if ok {
			return paramValStr
		}
	}
	return ""
}

func (m *MTAResolver) getParameter(sourceModule *mta.Module, source *mtaSource, requires *mta.Requires, paramName string) string {
	//first on source parameters scope
	paramValStr := m.getParameterFromSource(source, paramName)

	//first on source parameters scope
	if paramValStr != "" {
		return paramValStr
	}

	//then try on requires level
	if requires != nil {
		paramVal := requires.Parameters[paramName]
		if paramVal != nil {
			return paramVal.(string)
		}
	}

	var ok bool
	if sourceModule != nil {
		paramVal := sourceModule.Parameters[paramName]
		if paramVal != nil {
			return paramVal.(string)
		}
		//defaults to context's module params:
		paramValStr, ok = m.context.modules[sourceModule.Name][paramName]
		if ok {
			return paramValStr
		}
	}

	//then on MTA root scope
	paramVal := m.Parameters[paramName]
	if paramVal != nil {
		return paramVal.(string)
	}

	//then global scope
	paramValStr, ok = m.context.global[paramName]
	if ok {
		return paramValStr
	}

	if source == nil {
		println("Missing ", paramName)
	} else {
		println("Missing ", source.Name+"/"+paramName)
	}

	return "${" + paramName + "}"
}

func (m *MTAResolver) findProvider(name string) *mtaSource {
	for _, module := range m.Modules {
		for _, provides := range module.Provides {
			if provides.Name == name {
				source := mtaSource{Name: module.Name, Properties: provides.Properties, Parameters: nil, Type: moduleType, Module: module}
				return &source
			}
		}
	}

	//in case of resource, its name is the matching to the requires name
	for _, resource := range m.Resources {
		if resource.Name == name {
			source := mtaSource{Name: resource.Name, Properties: resource.Properties, Parameters: resource.Parameters, Type: resourceType, Resource: resource}
			return &source
		}

	}
	return nil
}

func convertToJSONSafe(val interface{}) interface{} {
	switch v := val.(type) {
	case map[interface{}]interface{}:
		res := map[string]interface{}{}
		for k, v := range v {
			res[fmt.Sprint(k)] = convertToJSONSafe(v)
		}
		return res
	case []interface{}:
		for k, v2 := range v {
			v[k] = convertToJSONSafe(v2)
		}
		return v
	}
	return val
}
