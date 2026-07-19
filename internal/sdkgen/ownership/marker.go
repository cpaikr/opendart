// Package ownership defines the shared generated-tree ownership marker.
package ownership

import "fmt"

const (
	Filename = ".opendart-sdk-generated"

	// MarkerPrefix identifies the generator schema field in every owned-tree marker.
	MarkerPrefix = "opendart-sdk-generator-schema="
)

// Marker returns the complete marker content for a generator model schema.
func Marker(schemaVersion uint32) string {
	return fmt.Sprintf("%s%d\n", MarkerPrefix, schemaVersion)
}
