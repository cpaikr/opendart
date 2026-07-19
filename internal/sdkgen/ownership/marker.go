// Package ownership defines the shared generated-tree ownership marker.
package ownership

import "fmt"

const Filename = ".opendart-sdk-generated"

// Marker returns the complete marker content for a generator model schema.
func Marker(schemaVersion uint32) string {
	return fmt.Sprintf("opendart-sdk-generator-schema=%d\n", schemaVersion)
}
