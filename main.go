package c3x

import _ "embed"

//go:embed c3x-usage-example.yml
var referenceUsageFileContents []byte

func GetReferenceUsageFileContents() *[]byte {
	return &referenceUsageFileContents
}
