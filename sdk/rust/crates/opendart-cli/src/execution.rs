use std::time::Duration;

use clap::ArgMatches;
use opendart::{ApiKey, Client, ClientBuilder, PreparedRequest, SourceReply, SourceResponse};
use serde::Serialize;

use crate::error::ErrorEnvelope;

const USER_AGENT_SUFFIX: &str = concat!("opendart-cli/", env!("CARGO_PKG_VERSION"));

#[derive(Clone, Copy, Serialize)]
pub(crate) struct OperationContext {
    name: &'static str,
    logical_id: &'static str,
    physical_id: &'static str,
    representation: &'static str,
}

impl OperationContext {
    pub(crate) const fn new(
        name: &'static str,
        logical_id: &'static str,
        physical_id: &'static str,
        representation: &'static str,
    ) -> Self {
        Self {
            name,
            logical_id,
            physical_id,
            representation,
        }
    }
}

pub(crate) struct ClientOverrides {
    connect_timeout_ms: Option<u64>,
    read_timeout_ms: Option<u64>,
    total_timeout_ms: Option<u64>,
    envelope_limit_bytes: Option<u64>,
}

impl ClientOverrides {
    pub(crate) fn from_matches(matches: &ArgMatches) -> Self {
        let matches = matches
            .subcommand()
            .map(|(_, matches)| matches)
            .unwrap_or(matches);
        Self {
            connect_timeout_ms: matches.get_one::<u64>("connect-timeout-ms").copied(),
            read_timeout_ms: matches.get_one::<u64>("read-timeout-ms").copied(),
            total_timeout_ms: matches.get_one::<u64>("total-timeout-ms").copied(),
            envelope_limit_bytes: matches.get_one::<u64>("envelope-limit-bytes").copied(),
        }
    }
}

pub(crate) struct BufferedOutput {
    pub(crate) bytes: Vec<u8>,
    pub(crate) exit: u8,
}

pub(crate) struct Executor {
    client: Client,
    runtime: tokio::runtime::Runtime,
}

impl Executor {
    pub(crate) fn new(
        key: ApiKey,
        overrides: ClientOverrides,
        operation: OperationContext,
    ) -> Result<Self, ErrorEnvelope> {
        let mut builder = Client::builder(key).user_agent_suffix(USER_AGENT_SUFFIX);
        if let Some(value) = overrides.connect_timeout_ms {
            builder = builder.connect_timeout(Duration::from_millis(value));
        }
        if let Some(value) = overrides.read_timeout_ms {
            builder = builder.read_timeout(Duration::from_millis(value));
        }
        if let Some(value) = overrides.total_timeout_ms {
            builder = builder.total_timeout(Duration::from_millis(value));
        }
        if let Some(value) = overrides.envelope_limit_bytes {
            let value = usize::try_from(value)
                .map_err(|_| ErrorEnvelope::invalid_client_setting(operation))?;
            builder = builder.envelope_limit(value);
        }
        let builder = compatibility_origin(builder)
            .map_err(|()| ErrorEnvelope::invalid_client_setting(operation))?;
        let client = builder
            .build()
            .map_err(|error| ErrorEnvelope::client_build(operation, error))?;
        let runtime = tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .build()
            .map_err(|_| ErrorEnvelope::client_initialization_for(operation))?;
        Ok(Self { client, runtime })
    }

    pub(crate) fn execute<T>(
        &self,
        request: PreparedRequest<T>,
        operation: OperationContext,
    ) -> Result<BufferedOutput, ErrorEnvelope>
    where
        T: Serialize,
    {
        let response = self
            .runtime
            .block_on(self.client.execute(&request))
            .map_err(|error| ErrorEnvelope::client(operation, error))?;
        encode_response(operation, response)
    }
}

#[derive(Serialize)]
struct ResponseEnvelope<T> {
    kind: &'static str,
    operation: OperationContext,
    response: T,
}

fn encode_response<T>(
    operation: OperationContext,
    response: SourceResponse<SourceReply<T>>,
) -> Result<BufferedOutput, ErrorEnvelope>
where
    T: Serialize,
{
    let exit = if matches!(&response.reply, SourceReply::Success(_)) {
        0
    } else {
        1
    };
    let envelope = ResponseEnvelope {
        kind: "response",
        operation,
        response,
    };
    let bytes =
        crate::output::encode(&envelope).map_err(|()| ErrorEnvelope::output_encode(operation))?;
    Ok(BufferedOutput { bytes, exit })
}

#[cfg(opendart_compat)]
fn compatibility_origin(builder: ClientBuilder) -> Result<ClientBuilder, ()> {
    match std::env::var("OPENDART_COMPAT_ORIGIN") {
        Ok(origin) => Ok(builder.__compatibility_origin(origin)),
        Err(std::env::VarError::NotPresent) => Ok(builder),
        Err(std::env::VarError::NotUnicode(_)) => Err(()),
    }
}

#[cfg(not(opendart_compat))]
fn compatibility_origin(builder: ClientBuilder) -> Result<ClientBuilder, ()> {
    Ok(builder)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn response_context_uses_the_public_contract_field_order() {
        let context = OperationContext::new("company", "DS001-2019002", "get_company_json", "json");
        assert_eq!(
            serde_json::to_string(&context).unwrap(),
            r#"{"name":"company","logical_id":"DS001-2019002","physical_id":"get_company_json","representation":"json"}"#
        );
    }
}
