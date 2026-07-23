use crate::generated::{GENERATOR_SCHEMA, PROJECTION_CHECKSUM};

const CANONICAL_BUNDLE_SHA256: &str =
    "230905dec268096f7d57a3b48cc2772b7bfcccf733ece124de9b7586f5f2d338";
const SPECIFICATION_SOURCE_RELEASE: Option<&str> = Some("v0.1.0");

/// The reviewed specification sources, generated artifact, and SDK projection.
///
/// The packaged archive's Cargo-generated `.cargo_vcs_info.json` records the
/// exact repository revision. The specification source release identifies the
/// canonical source inputs semantically; the bundle checksum independently
/// identifies the exact generated OpenAPI artifact selected for this crate.
#[derive(Clone, Copy, Debug, Eq, Hash, PartialEq)]
pub struct SourceProvenance {
    crate_version: &'static str,
    specification_source_release: Option<&'static str>,
    canonical_bundle_sha256: &'static str,
    generator_schema: u32,
    sdk_projection_sha256: &'static str,
}

impl SourceProvenance {
    /// Returns the Cargo package version.
    #[must_use]
    pub const fn crate_version(self) -> &'static str {
        self.crate_version
    }

    /// Returns the release whose canonical specification sources were selected.
    ///
    /// This tag identifies source inputs, not byte identity of a generated
    /// bundle. Use [`Self::canonical_bundle_sha256`] for exact artifact identity.
    #[must_use]
    pub const fn specification_source_release(self) -> Option<&'static str> {
        self.specification_source_release
    }

    /// Returns the SHA-256 of the selected canonical OpenAPI bundle.
    #[must_use]
    pub const fn canonical_bundle_sha256(self) -> &'static str {
        self.canonical_bundle_sha256
    }

    /// Returns the private normalized-model schema version.
    #[must_use]
    pub const fn generator_schema(self) -> u32 {
        self.generator_schema
    }

    /// Returns the deterministic generated Rust projection SHA-256.
    #[must_use]
    pub const fn sdk_projection_sha256(self) -> &'static str {
        self.sdk_projection_sha256
    }
}

/// Returns the source snapshot implemented by this crate version.
#[must_use]
pub const fn source_provenance() -> SourceProvenance {
    SourceProvenance {
        crate_version: env!("CARGO_PKG_VERSION"),
        specification_source_release: SPECIFICATION_SOURCE_RELEASE,
        canonical_bundle_sha256: CANONICAL_BUNDLE_SHA256,
        generator_schema: GENERATOR_SCHEMA,
        sdk_projection_sha256: PROJECTION_CHECKSUM,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn release_snapshot_has_complete_stable_identity() {
        let provenance = source_provenance();
        assert_eq!(provenance.crate_version(), env!("CARGO_PKG_VERSION"));
        assert_eq!(provenance.specification_source_release(), Some("v0.1.0"));
        assert_eq!(provenance.canonical_bundle_sha256().len(), 64);
        assert_eq!(provenance.sdk_projection_sha256().len(), 64);
        assert_ne!(
            provenance.canonical_bundle_sha256(),
            provenance.sdk_projection_sha256()
        );
    }
}
