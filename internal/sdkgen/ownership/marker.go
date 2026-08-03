// Package ownership defines the generated CLI projection ownership markers.
package ownership

import "fmt"

const (
	// CLIInterfaceFilename marks the public grammar and discovery projection.
	CLIInterfaceFilename = ".opendart-cli-interface-generated"
	// CLIDispatchFilename marks the private handwritten-SDK adapter projection.
	CLIDispatchFilename = ".opendart-cli-dispatch-generated"

	// CLIInterfaceMarkerPrefix identifies the interface projection schema.
	CLIInterfaceMarkerPrefix = "opendart-cli-interface-generator-schema="
	// CLIDispatchMarkerPrefix identifies the dispatch projection schema.
	CLIDispatchMarkerPrefix = "opendart-cli-dispatch-generator-schema="
)

// CLIInterfaceMarker returns the complete public interface ownership marker.
func CLIInterfaceMarker(schemaVersion uint32) string {
	return fmt.Sprintf("%s%d\n", CLIInterfaceMarkerPrefix, schemaVersion)
}

// CLIDispatchMarker returns the complete private adapter ownership marker.
func CLIDispatchMarker(schemaVersion uint32) string {
	return fmt.Sprintf("%s%d\n", CLIDispatchMarkerPrefix, schemaVersion)
}
