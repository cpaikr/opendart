const CANONICAL_BUNDLE_SHA256: &str =
    "61dae078d750cde76a83ccb48d5b37ab9bf9d034528fd46600aff1d2523e34e3";
const SPECIFICATION_SOURCE_RELEASE: Option<&str> = Some("v0.1.0");

/// The reviewed specification sources implemented by this crate.
///
/// The packaged archive's Cargo-generated `.cargo_vcs_info.json` records the
/// exact repository revision. The specification source release identifies the
/// canonical source inputs semantically; the bundle checksum independently
/// identifies the exact canonical OpenAPI bundle selected for this crate.
#[derive(Clone, Copy, Debug, Eq, Hash, PartialEq)]
pub struct SourceProvenance {
    crate_version: &'static str,
    specification_source_release: Option<&'static str>,
    canonical_bundle_sha256: &'static str,
}

impl SourceProvenance {
    /// Returns the Cargo package version.
    #[must_use]
    pub const fn crate_version(self) -> &'static str {
        self.crate_version
    }

    /// Returns the release whose canonical specification sources were selected.
    ///
    /// This tag identifies source inputs, not byte identity of the canonical
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
}

/// Returns the source snapshot implemented by this crate version.
#[must_use]
pub const fn source_provenance() -> SourceProvenance {
    SourceProvenance {
        crate_version: env!("CARGO_PKG_VERSION"),
        specification_source_release: SPECIFICATION_SOURCE_RELEASE,
        canonical_bundle_sha256: CANONICAL_BUNDLE_SHA256,
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
    }
}
