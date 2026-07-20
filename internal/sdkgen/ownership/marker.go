// Package ownership defines the shared generated-tree ownership marker.
package ownership

import "fmt"

const (
	Filename = ".opendart-sdk-generated"
	// CLIFilename marks the independently owned generated CLI tree.
	CLIFilename = ".opendart-cli-generated"

	// MarkerPrefix identifies the generator schema field in every owned-tree marker.
	MarkerPrefix = "opendart-sdk-generator-schema="
	// CLIMarkerPrefix identifies the CLI projection schema in its owned tree.
	CLIMarkerPrefix = "opendart-cli-generator-schema="
)

// Marker returns the complete marker content for a generator model schema.
func Marker(schemaVersion uint32) string {
	return fmt.Sprintf("%s%d\n", MarkerPrefix, schemaVersion)
}

// CLIMarker returns the complete marker content for the generated CLI tree.
func CLIMarker(schemaVersion uint32) string {
	return fmt.Sprintf("%s%d\n", CLIMarkerPrefix, schemaVersion)
}
