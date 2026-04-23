package docs

import (
	"os"
	"text/template"

	"github.com/c3xdev/c3x/internal/cloud/terraform"
)

func generateSupportedResourcesDocs(docsTemplatesPath string, outputPath string) error {
	tmpl, err := template.ParseFiles(docsTemplatesPath + "/supported_resources.md")
	if err != nil {
		return err
	}
	f, err := os.Create(outputPath + "/supported_resources.md")
	if err != nil {
		return err
	}
	defer f.Close()
	err = tmpl.Execute(f, terraform.ResourceRegistryMap)
	if err != nil {
		return err
	}
	return nil
}

func GenerateDocs(docsTemplatesPath, outputPath string) error {
	err := os.MkdirAll(outputPath, os.ModePerm)
	if err != nil {
		return err
	}
	err = generateSupportedResourcesDocs(docsTemplatesPath, outputPath)
	if err != nil {
		return err
	}
	return nil
}
