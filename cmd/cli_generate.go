package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
)

func runCLIGenerate(args []string) error {
	if len(args) == 0 {
		printCLIGenerateUsage(os.Stdout)
		return nil
	}

	kind := strings.ToLower(strings.TrimSpace(args[0]))
	if isHelpArg(kind) {
		printCLIGenerateUsage(os.Stdout)
		return nil
	}

	xargs := args[1:]

	switch kind {
	case "action":
		return generateAction(xargs)
	case "api":
		return generateAPI(xargs)
	case "model":
		return generateModel(xargs)
	case "migration":
		return generateMigration(xargs)
	case "service":
		return generateService(xargs)
	case "cmd":
		return generateCmd(xargs)
	default:
		return fmt.Errorf("unknown generator: %s", kind)
	}
}

func generateAction(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCLIGenerateActionUsage(os.Stdout)
		return nil
	}

	if len(args) != 2 {
		return fmt.Errorf("usage: airway cli generate action [api] [action]")
	}

	mod := strings.TrimSpace(args[0])
	name := strings.TrimSpace(args[1])
	apiName := apiDirName(mod)

	targetFile := filepath.Join(".", "app", "api", apiName, name+"_action.go")
	data := actionTemplateData{
		Name:    toCamelName(name),
		APIName: apiName,
	}

	return writeTemplateFile(actionTemplate, targetFile, data)
}

func generateAPI(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCLIGenerateAPIUsage(os.Stdout)
		return nil
	}

	if len(args) != 1 {
		return fmt.Errorf("usage: airway cli generate api [name]")
	}

	name := strings.TrimSpace(args[0])
	apiName := apiDirName(name)
	dirPath := filepath.Join(".", "app", "api", apiName)
	if err := ensureDir(dirPath); err != nil {
		return err
	}

	if err := generateAction([]string{name, "index"}); err != nil {
		return err
	}

	return writeTemplateFile(
		routesTemplate,
		filepath.Join(dirPath, "routes.go"),
		routesTemplateData{Mod: name, APIName: apiName},
	)
}

func generateModel(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCLIGenerateModelUsage(os.Stdout)
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("usage: airway cli generate model [name] [field:type]...")
	}

	name := strings.TrimSpace(args[0])
	lowerName := strings.ToLower(name)
	targetFile := filepath.Join(".", "app", "models", lowerName+".go")

	return writeTemplateFile(modelTemplate, targetFile, modelTemplateData{
		Name:      toCamelName(lowerName),
		TableName: lowerName + "s",
	})
}

func generateMigration(args []string) error {
	return generateMigrationFiles(args)
}

func generateService(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCLIGenerateServiceUsage(os.Stdout)
		return nil
	}

	if len(args) < 2 {
		return fmt.Errorf("usage: airway cli generate service <name> <field:type> <field:type>...")
	}

	name := strings.TrimSpace(args[0])
	data := serviceTemplateData{Name: toCamelName(name)}

	fields := make([]string, 0, len(args)-1)
	assignments := make([]string, 0, len(args)-1)
	for _, arg := range args[1:] {
		fieldName, fieldType, err := parseFieldArg(arg)
		if err != nil {
			return err
		}

		fields = append(fields, fmt.Sprintf("%s %s", fieldName, fieldType))
		assignments = append(assignments, fmt.Sprintf("%q: %s", fieldName, fieldName))
	}

	data.Fields = strings.Join(fields, ", ")
	data.SQLH = "sql.H{" + strings.Join(assignments, ", ") + "}"

	return writeTemplateFile(
		serviceTemplate,
		filepath.Join(".", "app", "services", name+".go"),
		data,
	)
}

func generateCmd(args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		printCLIGenerateCmdUsage(os.Stdout)
		return nil
	}

	if len(args) < 2 {
		return fmt.Errorf("usage: airway cli generate cmd <name> <field> <field>...")
	}

	name := strings.TrimSpace(args[0])
	data := cmdTemplateData{
		Name:      toCamelName(name),
		LowerName: name,
	}

	fields := make([]string, 0, len(args)-1)
	callArgs := make([]string, 0, len(args)-1)
	updateArgs := make([]string, 0, len(args)-1)

	for index, arg := range args[1:] {
		fieldName := strings.TrimSpace(arg)
		if fieldName == "" {
			return fmt.Errorf("command field must not be empty")
		}

		fields = append(fields, fmt.Sprintf("<%s>", fieldName))
		callArgs = append(callArgs, fmt.Sprintf("args[%d]", index))
		updateArgs = append(updateArgs, fmt.Sprintf("args[%d]", index+1))
	}

	data.Fields = strings.Join(fields, " ")
	data.Args = strings.Join(callArgs, ", ")
	data.Args1 = strings.Join(updateArgs, ", ")
	data.CreateArgCount = len(args) - 1
	data.UpdateArgCount = len(args)

	return writeTemplateFile(
		cmdTemplate,
		filepath.Join(".", "cmd", name+".go"),
		data,
	)
}

func apiDirName(name string) string {
	return fmt.Sprintf("%s_api", strings.TrimSpace(name))
}

func toCamelName(value string) string {
	camel := strcase.ToCamel(strings.TrimSpace(value))
	camel = strings.ReplaceAll(camel, "Uuid", "UUID")
	camel = strings.ReplaceAll(camel, "Url", "URL")
	camel = strings.ReplaceAll(camel, "Api", "API")
	return camel
}

func parseFieldArg(arg string) (string, string, error) {
	fieldName, fieldType, found := strings.Cut(strings.TrimSpace(arg), ":")
	if !found || strings.TrimSpace(fieldName) == "" || strings.TrimSpace(fieldType) == "" {
		return "", "", fmt.Errorf("invalid field %q, expected name:type", arg)
	}

	return strings.TrimSpace(fieldName), strings.TrimSpace(fieldType), nil
}
