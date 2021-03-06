package validate

import (
	"gopkg.in/yaml.v3"
	"strings"

	"github.com/pkg/errors"

	"github.com/SAP/cloud-mta/mta"
)

const (
	couldNotValidateErrorMsg = `could not validate the "%s" file`
	validationErrorsMsg      = "the \"%s\" file is not valid:\n%s"
)

// Mtaext validates an MTA extension file.
func Mtaext(projectPath, extPath string,
	validateSchema, validateSemantic, strict bool, exclude string) (warning string, err error) {
	if validateSemantic || validateSchema {
		var errIssues, warnIssues YamlValidationIssues

		// ParseFile contains MTA yaml content.
		yamlContent, e := readFile(extPath)

		if e != nil {
			return "", errors.Wrapf(e, couldNotValidateErrorMsg, extPath)
		}
		s := string(yamlContent)
		s = strings.Replace(s, "\r\n", "\r", -1)
		yamlContent = []byte(s)

		// Validates MTA content.
		contentErrIssues, contentWarnIssues := validateExt(yamlContent, projectPath, extPath,
			validateSchema, validateSemantic, strict, exclude)
		errIssues = append(errIssues, contentErrIssues...)
		errIssues.Sort()
		warnIssues = append(warnIssues, contentWarnIssues...)
		warnIssues.Sort()
		if len(errIssues) > 0 {
			return warnIssues.String(), errors.Errorf(validationErrorsMsg, extPath, errIssues.String())
		}
		return warnIssues.String(), nil
	}

	return "", nil
}

// validateExt validates the MTA extension descriptor
func validateExt(yamlContent []byte, projectPath string, extFileName string,
	validateSchema, validateSemantic, strict bool, exclude string) (errIssues YamlValidationIssues, warnIssues YamlValidationIssues) {
	extNode, err := getContentNode(yamlContent)
	if err != nil {
		errIssues = convertError(err)
	}

	// Errors from this unmarshal include those from getContentNode and have more validations, so they should override
	// the previous errors
	mtaExt, err := mta.UnmarshalExt(yamlContent)
	if strict && err != nil {
		errIssues = convertError(err)
	} else if err != nil {
		warnIssues = convertError(err)
	}

	if validateSchema {
		errs, warns := validateExtSchema(mtaExt, extNode, strict)
		errIssues = append(errIssues, errs...)
		warnIssues = append(warnIssues, warns...)
	}

	if validateSemantic {
		errs, warns := runExtSemanticValidations(mtaExt, extNode, projectPath, exclude, strict)
		errIssues = append(errIssues, errs...)
		warnIssues = append(warnIssues, warns...)
	}
	return errIssues, warnIssues
}

func validateExtSchema(mtaExt *mta.EXT, extNode *yaml.Node, strict bool) (errIssues YamlValidationIssues, warnIssues YamlValidationIssues) {
	validations, schemaValidationLog := buildValidationsFromSchemaText(extSchemaDef)
	if len(schemaValidationLog) > 0 {
		errIssues = append(errIssues, schemaValidationLog...)
		return errIssues, warnIssues
	}
	errIssues = append(errIssues, runSchemaValidations(extNode, validations...)...)

	issues := runAdditionalExtSchemaValidations(mtaExt, extNode, "")
	if strict {
		errIssues = append(errIssues, issues...)
	} else {
		warnIssues = append(warnIssues, issues...)
	}
	return errIssues, warnIssues
}

func runAdditionalExtSchemaValidations(mtaExt *mta.EXT, extNode *yaml.Node, source string) []YamlValidationIssue {
	requiresCheck := property(requiresYamlField, forEach(
		propertyName(listYamlField, doesNotExist()),
		propertyName(propertiesMetadataField, doesNotExist()),
		propertyName(parametersMetadataField, doesNotExist()),
	))
	return runSchemaValidations(extNode, sequence(
		property(modulesYamlField, forEach(
			property(providesYamlField, forEach(
				propertyName(publicYamlField, doesNotExist()),
			)),
			requiresCheck,
			property(hooksYamlField, forEach(requiresCheck)),
		)),
		property(resourcesYamlField, forEach(requiresCheck)),
	))
}
