//! Repository-only bridge into the exact private client factory.

use crate::client::{Client, ClientBuilder};

/// Returns the exact private HTTP adapter for observable compatibility tests.
pub fn http_client(client: &Client) -> reqwest::Client {
    client.compatibility_http_client()
}

impl ClientBuilder {
    /// Overrides the production origin only for this isolated compatibility
    /// fixture. This module is outside the published crate source tree.
    #[doc(hidden)]
    pub fn __compatibility_origin(self, origin: String) -> Self {
        self.compatibility_origin(origin)
    }
}
