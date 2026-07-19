use crate::generated::{GENERATOR_SCHEMA, PROJECTION_CHECKSUM};

const CANONICAL_BUNDLE_SHA256: &str =
    "76711f2e9c886eb1977f1292c07fcefdd60528a910982cc92b2df565ab97fe24";
const SPECIFICATION_RELEASE: Option<&str> = Some("v0.1.0");

/// The reviewed specification and generator snapshot implemented by this crate.
///
/// The packaged archive's Cargo-generated `.cargo_vcs_info.json` records the
/// exact source revision. These values identify the independent specification
/// and generated-contract inputs selected for the crate release.
#[derive(Clone, Copy, Debug, Eq, Hash, PartialEq)]
pub struct SourceProvenance {
    crate_version: &'static str,
    specification_release: Option<&'static str>,
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

    /// Returns the selected specification release tag, when one exists.
    #[must_use]
    pub const fn specification_release(self) -> Option<&'static str> {
        self.specification_release
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
        specification_release: SPECIFICATION_RELEASE,
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
        assert!(provenance.specification_release().is_some());
        assert_eq!(provenance.canonical_bundle_sha256().len(), 64);
        assert_eq!(provenance.sdk_projection_sha256().len(), 64);
        assert_ne!(
            provenance.canonical_bundle_sha256(),
            provenance.sdk_projection_sha256()
        );
    }
}
