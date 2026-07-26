use std::fs;
use std::future::{Future, poll_fn};
use std::io::{self, Write};
use std::path::{Path, PathBuf};
use std::pin::Pin;
use std::time::Duration;

use clap::ArgMatches;
use futures_core::Stream;
use opendart::{
    BinaryReply, BodyStream, Client, ClientError, PreparedBinaryRequest, ResponseMetadata,
    SourceResponse, StatusEnvelope, TransportFailureKind,
};
use serde::Serialize;

use crate::error::{ArtifactIoReason, CleanupContext, ErrorEnvelope};
use crate::execution::{BufferedOutput, OperationContext};

mod transaction;

use transaction::{ArtifactTransaction, FailureKind, WorkerError};

pub(crate) enum TargetError {
    Usage(ErrorEnvelope),
    Execution(ErrorEnvelope),
}

pub(crate) struct ArtifactTarget {
    path: PathBuf,
    spelling: String,
    limit: u64,
}

impl ArtifactTarget {
    pub(crate) fn from_matches(
        matches: &ArgMatches,
        default_limit: u64,
        operation: OperationContext,
    ) -> Result<Self, TargetError> {
        let matches = matches
            .subcommand()
            .map(|(_, matches)| matches)
            .unwrap_or(matches);
        let spelling = matches
            .get_one::<String>("output")
            .expect("clap requires binary output")
            .clone();
        if spelling.is_empty() || spelling == "-" {
            return Err(TargetError::Usage(ErrorEnvelope::invalid_artifact_output(
                operation,
            )));
        }
        let path = PathBuf::from(&spelling);
        match fs::symlink_metadata(&path) {
            Ok(_) => {
                return Err(TargetError::Execution(ErrorEnvelope::destination_exists(
                    operation, None, spelling,
                )));
            }
            Err(error)
                if matches!(
                    error.kind(),
                    io::ErrorKind::NotFound | io::ErrorKind::NotADirectory
                ) => {}
            Err(_) => {
                return Err(TargetError::Execution(ErrorEnvelope::artifact_io(
                    operation,
                    None,
                    spelling,
                    ArtifactIoReason::DestinationMetadataUnavailable,
                )));
            }
        }
        let parent = artifact_parent(&path);
        match fs::metadata(parent) {
            Ok(metadata) if metadata.is_dir() => {}
            Ok(_) => {
                return Err(TargetError::Execution(ErrorEnvelope::artifact_io(
                    operation,
                    None,
                    spelling,
                    ArtifactIoReason::ParentNotDirectory,
                )));
            }
            Err(_) => {
                return Err(TargetError::Execution(ErrorEnvelope::artifact_io(
                    operation,
                    None,
                    spelling,
                    ArtifactIoReason::ParentUnavailable,
                )));
            }
        }
        let limit = matches
            .get_one::<u64>("artifact-limit-bytes")
            .copied()
            .unwrap_or(default_limit);
        Ok(Self {
            path,
            spelling,
            limit,
        })
    }
}

fn artifact_parent(path: &Path) -> &Path {
    path.parent()
        .filter(|parent| !parent.as_os_str().is_empty())
        .unwrap_or_else(|| Path::new("."))
}

pub(crate) async fn execute(
    client: &Client,
    request: PreparedBinaryRequest,
    operation: OperationContext,
    target: ArtifactTarget,
    total_timeout: Duration,
) -> Result<BufferedOutput, ErrorEnvelope> {
    let spelling = target.spelling.clone();
    let staged = ArtifactTransaction::start(target)
        .await
        .map_err(|error| worker_error(operation, None, &spelling, error))?;
    let deadline = tokio::time::Instant::now()
        .checked_add(total_timeout)
        .expect("the SDK validates its configured total timeout");
    let response = match client.execute_binary(&request).await {
        Ok(response) => response,
        Err(error) => {
            let timeout = matches!(
                &error,
                ClientError::Transport(error)
                    if matches!(error.kind(), TransportFailureKind::Timeout)
            );
            let fallback = ErrorEnvelope::client(operation, error);
            let cleanup = if timeout {
                Some(staged.abandon())
            } else {
                staged.discard().await
            };
            return Err(fallback.with_cleanup(cleanup));
        }
    };
    let SourceResponse {
        metadata, reply, ..
    } = response;
    match reply {
        BinaryReply::Archive(stream) => {
            stream_to_artifact(
                operation,
                metadata,
                stream,
                staged,
                spelling,
                ArtifactKind::Archive,
                deadline,
            )
            .await
        }
        BinaryReply::Status(status) => {
            let cleanup = staged.discard().await;
            encode_report(
                operation,
                &metadata,
                cleanup,
                ArtifactReply::Status(&status),
                1,
            )
            .map_err(|error| error.with_cleanup(cleanup))
        }
        BinaryReply::Unrecognized(stream) => {
            stream_to_artifact(
                operation,
                metadata,
                stream,
                staged,
                spelling,
                ArtifactKind::Unrecognized,
                deadline,
            )
            .await
        }
        _ => {
            let fallback = ErrorEnvelope::sdk_contract_mismatch(Some(operation));
            Err(fallback.with_cleanup(staged.discard().await))
        }
    }
}

#[derive(Clone, Copy)]
enum ArtifactKind {
    Archive,
    Unrecognized,
}

async fn stream_to_artifact(
    operation: OperationContext,
    metadata: ResponseMetadata,
    mut stream: BodyStream,
    mut staged: ArtifactTransaction,
    spelling: String,
    kind: ArtifactKind,
    deadline: tokio::time::Instant,
) -> Result<BufferedOutput, ErrorEnvelope> {
    while let Some(chunk) = poll_fn(|context| Pin::new(&mut stream).poll_next(context)).await {
        let chunk = match chunk {
            Ok(chunk) => chunk,
            Err(error) => {
                let timeout = matches!(error.kind(), TransportFailureKind::Timeout);
                let fallback =
                    ErrorEnvelope::body_stream(operation, metadata.clone(), error.kind());
                let cleanup = if timeout {
                    Some(staged.abandon())
                } else {
                    staged.discard().await
                };
                return Err(fallback.with_cleanup(cleanup));
            }
        };
        match before_deadline(deadline, staged.enqueue(chunk.as_bytes().to_vec())).await {
            Ok(Ok(())) => {}
            Ok(Err(error)) => {
                return Err(worker_error(operation, Some(metadata), &spelling, error));
            }
            Err(()) => {
                let fallback =
                    ErrorEnvelope::body_stream(operation, metadata, TransportFailureKind::Timeout);
                return Err(fallback.with_cleanup(Some(staged.abandon())));
            }
        }
    }
    let bytes = match before_deadline(deadline, staged.finish()).await {
        Ok(Ok(bytes)) => bytes,
        Ok(Err(error)) => {
            return Err(worker_error(
                operation,
                Some(metadata.clone()),
                &spelling,
                error,
            ));
        }
        Err(()) => {
            let fallback =
                ErrorEnvelope::body_stream(operation, metadata, TransportFailureKind::Timeout);
            return Err(fallback.with_cleanup(Some(staged.abandon())));
        }
    };

    let reference = ArtifactReference {
        path: &spelling,
        bytes,
    };
    let (reply, exit) = match kind {
        ArtifactKind::Archive => (ArtifactReply::Archive(reference), 0),
        ArtifactKind::Unrecognized => (ArtifactReply::Unrecognized(reference), 1),
    };
    let output = match encode_report(operation, &metadata, None, reply, exit) {
        Ok(output) => output,
        Err(error) => {
            return Err(error.with_cleanup(staged.discard().await));
        }
    };
    let cleanup_output = match encode_report(
        operation,
        &metadata,
        Some(CleanupContext::discard_staging_link()),
        reply,
        exit,
    ) {
        Ok(output) => output,
        Err(error) => return Err(error.with_cleanup(staged.discard().await)),
    };

    match staged.commit().await {
        Ok(Some(_)) => Ok(cleanup_output),
        Ok(None) => Ok(output),
        Err(error) => Err(worker_error(operation, Some(metadata), &spelling, error)),
    }
}

async fn before_deadline<T>(
    deadline: tokio::time::Instant,
    future: impl Future<Output = T>,
) -> Result<T, ()> {
    tokio::time::timeout_at(deadline, future)
        .await
        .map_err(|_| ())
}

fn worker_error(
    operation: OperationContext,
    metadata: Option<ResponseMetadata>,
    spelling: &str,
    error: WorkerError,
) -> ErrorEnvelope {
    let envelope = match error.kind {
        FailureKind::DestinationExists => {
            ErrorEnvelope::destination_exists(operation, metadata, spelling.to_owned())
        }
        FailureKind::DestinationMetadataUnavailable => ErrorEnvelope::artifact_io(
            operation,
            metadata,
            spelling.to_owned(),
            ArtifactIoReason::DestinationMetadataUnavailable,
        ),
        FailureKind::ParentNotDirectory => ErrorEnvelope::artifact_io(
            operation,
            metadata,
            spelling.to_owned(),
            ArtifactIoReason::ParentNotDirectory,
        ),
        FailureKind::ParentUnavailable => ErrorEnvelope::artifact_io(
            operation,
            metadata,
            spelling.to_owned(),
            ArtifactIoReason::ParentUnavailable,
        ),
        FailureKind::TemporaryFileCreation => ErrorEnvelope::artifact_io(
            operation,
            metadata,
            spelling.to_owned(),
            ArtifactIoReason::TemporaryFileCreation,
        ),
        FailureKind::Limit => match metadata {
            Some(metadata) => ErrorEnvelope::artifact_limit(operation, metadata),
            None => ErrorEnvelope::artifact_io(
                operation,
                None,
                spelling.to_owned(),
                ArtifactIoReason::WriteFailed,
            ),
        },
        FailureKind::Write | FailureKind::WorkerUnavailable => ErrorEnvelope::artifact_io(
            operation,
            metadata,
            spelling.to_owned(),
            ArtifactIoReason::WriteFailed,
        ),
        FailureKind::Flush => ErrorEnvelope::artifact_io(
            operation,
            metadata,
            spelling.to_owned(),
            ArtifactIoReason::FlushFailed,
        ),
        FailureKind::Publish => ErrorEnvelope::artifact_io(
            operation,
            metadata,
            spelling.to_owned(),
            ArtifactIoReason::PublishFailed,
        ),
    };
    envelope.with_cleanup(error.cleanup)
}

fn next_byte_count(current: u64, chunk: u64, limit: u64) -> Option<u64> {
    current.checked_add(chunk).filter(|next| *next <= limit)
}

#[derive(Debug, Eq, PartialEq)]
enum ArtifactWriteError {
    Limit,
    Io,
}

fn write_bounded(
    writer: &mut impl Write,
    current: u64,
    chunk: &[u8],
    limit: u64,
) -> Result<u64, ArtifactWriteError> {
    let chunk_len = u64::try_from(chunk.len()).map_err(|_| ArtifactWriteError::Limit)?;
    let next = next_byte_count(current, chunk_len, limit).ok_or(ArtifactWriteError::Limit)?;
    writer
        .write_all(chunk)
        .map_err(|_| ArtifactWriteError::Io)?;
    Ok(next)
}

fn encode_report(
    operation: OperationContext,
    metadata: &ResponseMetadata,
    cleanup: Option<CleanupContext>,
    reply: ArtifactReply<'_>,
    exit: u8,
) -> Result<BufferedOutput, ErrorEnvelope> {
    let envelope = BinaryResponseEnvelope {
        kind: "response",
        operation,
        cleanup,
        response: BinaryResponse { metadata, reply },
    };
    let bytes = crate::output::encode(&envelope)
        .map_err(|()| ErrorEnvelope::output_encode_with_metadata(operation, metadata.clone()))?;
    Ok(BufferedOutput { bytes, exit })
}

#[derive(Serialize)]
struct BinaryResponseEnvelope<'a> {
    kind: &'static str,
    operation: OperationContext,
    #[serde(skip_serializing_if = "Option::is_none")]
    cleanup: Option<CleanupContext>,
    response: BinaryResponse<'a>,
}

#[derive(Serialize)]
struct BinaryResponse<'a> {
    metadata: &'a ResponseMetadata,
    reply: ArtifactReply<'a>,
}

#[derive(Clone, Copy, Serialize)]
#[serde(tag = "kind", content = "value", rename_all = "snake_case")]
enum ArtifactReply<'a> {
    Archive(ArtifactReference<'a>),
    Status(&'a StatusEnvelope),
    Unrecognized(ArtifactReference<'a>),
}

#[derive(Clone, Copy, Serialize)]
struct ArtifactReference<'a> {
    path: &'a str,
    bytes: u64,
}

#[cfg(test)]
mod tests {
    use std::io::{self, Write};

    use super::{ArtifactWriteError, next_byte_count, write_bounded};

    #[test]
    fn inclusive_artifact_count_handles_the_default_boundary_and_overflow() {
        const DEFAULT_LIMIT: u64 = 536_870_912;

        assert_eq!(
            next_byte_count(DEFAULT_LIMIT - 1, 1, DEFAULT_LIMIT),
            Some(DEFAULT_LIMIT)
        );
        assert_eq!(next_byte_count(DEFAULT_LIMIT, 1, DEFAULT_LIMIT), None);
        assert_eq!(next_byte_count(u64::MAX, 1, u64::MAX), None);
    }

    #[test]
    fn bounded_writes_reject_overflow_before_io_and_classify_write_failure() {
        let mut bytes = Vec::new();
        assert_eq!(write_bounded(&mut bytes, 0, b"exact", 5), Ok(5));
        assert_eq!(bytes, b"exact");
        assert_eq!(
            write_bounded(&mut bytes, 5, b"overflow", 5),
            Err(ArtifactWriteError::Limit)
        );
        assert_eq!(bytes, b"exact");

        let mut writer = FailingWriter;
        assert_eq!(
            write_bounded(&mut writer, 0, b"body", 4),
            Err(ArtifactWriteError::Io)
        );
    }

    struct FailingWriter;

    impl Write for FailingWriter {
        fn write(&mut self, _buffer: &[u8]) -> io::Result<usize> {
            Err(io::Error::other("fixture"))
        }

        fn flush(&mut self) -> io::Result<()> {
            Ok(())
        }
    }
}
