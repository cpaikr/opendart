// Package ownership defines the shared generated-tree ownership marker.
package ownership

import "fmt"

const (
	Filename = ".opendart-sdk-generated"
	// CLIInterfaceFilename marks the public grammar and discovery projection.
	CLIInterfaceFilename = ".opendart-cli-interface-generated"
	// CLIDispatchFilename marks the private generated-SDK adapter projection.
	CLIDispatchFilename = ".opendart-cli-dispatch-generated"

	// MarkerPrefix identifies the generator schema field in every owned-tree marker.
	MarkerPrefix = "opendart-sdk-generator-schema="
	// CLIInterfaceMarkerPrefix identifies the interface projection schema.
	CLIInterfaceMarkerPrefix = "opendart-cli-interface-generator-schema="
	// CLIDispatchMarkerPrefix identifies the dispatch projection schema.
	CLIDispatchMarkerPrefix = "opendart-cli-dispatch-generator-schema="
)

// Marker returns the complete marker content for a generator model schema.
func Marker(schemaVersion uint32) string {
	return fmt.Sprintf("%s%d\n", MarkerPrefix, schemaVersion)
}

// CLIInterfaceMarker returns the complete public interface ownership marker.
func CLIInterfaceMarker(schemaVersion uint32) string {
	return fmt.Sprintf("%s%d\n", CLIInterfaceMarkerPrefix, schemaVersion)
}

// CLIDispatchMarker returns the complete private adapter ownership marker.
func CLIDispatchMarker(schemaVersion uint32) string {
	return fmt.Sprintf("%s%d\n", CLIDispatchMarkerPrefix, schemaVersion)
}
