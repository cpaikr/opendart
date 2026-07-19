//! Stable-only compatibility proof for adversarial `reqwest` feature unification.

use opendart::{ApiKey, Client, ClientBuildError};

/// Builds the official client while all transport-affecting dependency features are unified.
pub fn build_official_client() -> Result<Client, ClientBuildError> {
    Client::builder(ApiKey::new("feature-unification-fixture").expect("nonempty fixture key"))
        .build()
}

#[cfg(test)]
mod tests {
    use std::{error::Error, time::Duration};

    use opendart::{ApiKey, Client, Representation, operations::Company};
    use tokio::{io::AsyncReadExt, net::TcpListener, task::JoinHandle};

    async fn start_client_hello_capture() -> (String, JoinHandle<Vec<u8>>) {
        let listener = TcpListener::bind("127.0.0.1:0").await.unwrap();
        let port = listener.local_addr().unwrap().port();
        let task = tokio::spawn(async move {
            let (mut socket, _) = tokio::time::timeout(Duration::from_secs(2), listener.accept())
                .await
                .expect("the client did not reach the TLS fixture")
                .unwrap();
            let mut header = [0; 5];
            socket.read_exact(&mut header).await.unwrap();
            let length = usize::from(u16::from_be_bytes([header[3], header[4]]));
            let mut record = header.to_vec();
            record.resize(5 + length, 0);
            socket.read_exact(&mut record[5..]).await.unwrap();
            record
        });
        (format!("https://localhost.:{port}"), task)
    }

    fn cipher_suites(client_hello: &[u8]) -> Vec<u16> {
        assert_eq!(client_hello.first(), Some(&22), "expected a TLS handshake");
        assert_eq!(client_hello.get(5), Some(&1), "expected a ClientHello");
        let mut offset = 9 + 2 + 32;
        let session_id_length = usize::from(client_hello[offset]);
        offset += 1 + session_id_length;
        let suites_length = usize::from(u16::from_be_bytes([
            client_hello[offset],
            client_hello[offset + 1],
        ]));
        offset += 2;
        client_hello[offset..offset + suites_length]
            .chunks_exact(2)
            .map(|suite| u16::from_be_bytes([suite[0], suite[1]]))
            .collect()
    }

    fn company_request() -> opendart::PreparedRequest {
        Company::new("00126380")
            .prepare(Representation::Json)
            .unwrap()
    }

    async fn resolution_error(client: &reqwest::Client, host: &str) -> String {
        let error = client
            .get(format!("http://{host}/"))
            .send()
            .await
            .unwrap_err();
        let mut leaf: &(dyn Error + 'static) = &error;
        while let Some(source) = leaf.source() {
            leaf = source;
        }
        leaf.to_string()
    }

    #[test]
    fn official_factory_builds_under_adversarial_feature_unification() {
        super::build_official_client().expect("the guarded official factory must remain valid");
    }

    #[tokio::test]
    async fn official_factory_uses_system_resolution_when_hickory_is_unified() {
        let official = Client::builder(ApiKey::new("resolver-fixture").unwrap())
            .connect_timeout(Duration::from_secs(1))
            .__compatibility_origin("http://127.0.0.1".to_owned())
            .build()
            .unwrap();
        let official = opendart::compatibility::http_client(&official);
        let system = reqwest::Client::builder()
            .no_proxy()
            .no_hickory_dns()
            .connect_timeout(Duration::from_millis(500))
            .timeout(Duration::from_secs(1))
            .build()
            .unwrap();
        let hickory = reqwest::Client::builder()
            .no_proxy()
            .connect_timeout(Duration::from_millis(500))
            .timeout(Duration::from_secs(1))
            .build()
            .unwrap();
        let invalid_host = format!("{}.invalid", "x".repeat(64));

        let official_error = resolution_error(&official, &invalid_host).await;
        let system_error = resolution_error(&system, &invalid_host).await;
        let hickory_error = resolution_error(&hickory, &invalid_host).await;
        assert_eq!(official_error, system_error);
        assert_ne!(official_error, hickory_error);
    }

    #[tokio::test]
    async fn official_factory_emits_a_rustls_client_hello_when_native_tls_is_unified() {
        let (origin, server) = start_client_hello_capture().await;
        let official = Client::builder(ApiKey::new("tls-fixture").unwrap())
            .connect_timeout(Duration::from_secs(1))
            .total_timeout(Duration::from_secs(2))
            .__compatibility_origin(origin)
            .build()
            .unwrap();
        let _ = official.execute(&company_request()).await;
        let official_suites = cipher_suites(&server.await.unwrap());

        let (origin, server) = start_client_hello_capture().await;
        let rustls = reqwest::Client::builder()
            .no_proxy()
            .no_hickory_dns()
            .tls_backend_rustls()
            .timeout(Duration::from_secs(2))
            .build()
            .unwrap();
        let _ = rustls.get(origin).send().await;
        let rustls_suites = cipher_suites(&server.await.unwrap());

        let (origin, server) = start_client_hello_capture().await;
        let native = reqwest::Client::builder()
            .no_proxy()
            .no_hickory_dns()
            .timeout(Duration::from_secs(2))
            .build()
            .unwrap();
        let _ = native.get(origin).send().await;
        let native_suites = cipher_suites(&server.await.unwrap());

        assert_eq!(official_suites, rustls_suites);
        assert_ne!(official_suites, native_suites);
    }
}
