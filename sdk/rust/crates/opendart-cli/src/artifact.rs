use std::fs;
use std::future::poll_fn;
use std::io::{self, Write};
use std::path::{Path, PathBuf};
use std::pin::Pin;

use clap::ArgMatches;
use futures_core::Stream;
use opendart::{
    BinaryReply, BodyStream, Client, PreparedBinaryRequest, ResponseMetadata, SourceResponse,
    StatusEnvelope,
};
use serde::Serialize;

use crate::error::{ArtifactIoReason, ErrorEnvelope};
use crate::execution::{BufferedOutput, OperationContext};

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

    pub(crate) fn stage(
        self,
        operation: OperationContext,
    ) -> Result<StagedArtifact, ErrorEnvelope> {
        let parent = artifact_parent(&self.path);
        let file = tempfile::NamedTempFile::new_in(parent).map_err(|_| {
            ErrorEnvelope::artifact_io(
                operation,
                None,
                self.spelling.clone(),
                ArtifactIoReason::TemporaryFileCreation,
            )
        })?;
        Ok(StagedArtifact {
            file,
            path: self.path,
            spelling: self.spelling,
            limit: self.limit,
        })
    }
}

fn artifact_parent(path: &Path) -> &Path {
    path.parent()
        .filter(|parent| !parent.as_os_str().is_empty())
        .unwrap_or_else(|| Path::new("."))
}

pub(crate) struct StagedArtifact {
    file: tempfile::NamedTempFile,
    path: PathBuf,
    spelling: String,
    limit: u64,
}

pub(crate) async fn execute(
    client: &Client,
    request: PreparedBinaryRequest,
    operation: OperationContext,
    staged: StagedArtifact,
) -> Result<BufferedOutput, ErrorEnvelope> {
    let response = match client.execute_binary(&request).await {
        Ok(response) => response,
        Err(error) => {
            let metadata = error.metadata().cloned();
            let fallback = ErrorEnvelope::client(operation, error);
            return Err(discard_error(
                staged.file,
                operation,
                metadata,
                &staged.spelling,
                fallback,
            ));
        }
    };
    let SourceResponse {
        metadata, reply, ..
    } = response;
    match reply {
        BinaryReply::Archive(stream) => {
            stream_to_artifact(operation, metadata, stream, staged, ArtifactKind::Archive).await
        }
        BinaryReply::Status(status) => {
            let spelling = staged.spelling;
            staged.file.close().map_err(|_| {
                ErrorEnvelope::artifact_io(
                    operation,
                    Some(metadata.clone()),
                    spelling,
                    ArtifactIoReason::CleanupFailed,
                )
            })?;
            encode_report(operation, &metadata, ArtifactReply::Status(&status), 1)
        }
        BinaryReply::Unrecognized(stream) => {
            stream_to_artifact(
                operation,
                metadata,
                stream,
                staged,
                ArtifactKind::Unrecognized,
            )
            .await
        }
        _ => {
            let fallback = ErrorEnvelope::sdk_contract_mismatch(Some(operation));
            Err(discard_error(
                staged.file,
                operation,
                Some(metadata),
                &staged.spelling,
                fallback,
            ))
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
    staged: StagedArtifact,
    kind: ArtifactKind,
) -> Result<BufferedOutput, ErrorEnvelope> {
    let StagedArtifact {
        mut file,
        path,
        spelling,
        limit,
    } = staged;
    let mut bytes = 0_u64;
    while let Some(chunk) = poll_fn(|context| Pin::new(&mut stream).poll_next(context)).await {
        let chunk = match chunk {
            Ok(chunk) => chunk,
            Err(error) => {
                let fallback =
                    ErrorEnvelope::body_stream(operation, metadata.clone(), error.kind());
                return Err(discard_error(
                    file,
                    operation,
                    Some(metadata),
                    &spelling,
                    fallback,
                ));
            }
        };
        bytes = match write_bounded(&mut file, bytes, chunk.as_bytes(), limit) {
            Ok(next) => next,
            Err(ArtifactWriteError::Limit) => {
                let fallback = ErrorEnvelope::artifact_limit(operation, metadata.clone());
                return Err(discard_error(
                    file,
                    operation,
                    Some(metadata),
                    &spelling,
                    fallback,
                ));
            }
            Err(ArtifactWriteError::Io) => {
                let fallback = ErrorEnvelope::artifact_io(
                    operation,
                    Some(metadata.clone()),
                    spelling.clone(),
                    ArtifactIoReason::WriteFailed,
                );
                return Err(discard_error(
                    file,
                    operation,
                    Some(metadata),
                    &spelling,
                    fallback,
                ));
            }
        };
    }
    if file.flush().is_err() {
        let fallback = ErrorEnvelope::artifact_io(
            operation,
            Some(metadata.clone()),
            spelling.clone(),
            ArtifactIoReason::FlushFailed,
        );
        return Err(discard_error(
            file,
            operation,
            Some(metadata),
            &spelling,
            fallback,
        ));
    }

    let reference = ArtifactReference {
        path: &spelling,
        bytes,
    };
    let (reply, exit) = match kind {
        ArtifactKind::Archive => (ArtifactReply::Archive(reference), 0),
        ArtifactKind::Unrecognized => (ArtifactReply::Unrecognized(reference), 1),
    };
    let output = match encode_report(operation, &metadata, reply, exit) {
        Ok(output) => output,
        Err(error) => {
            return Err(discard_error(
                file,
                operation,
                Some(metadata),
                &spelling,
                error,
            ));
        }
    };

    match file.persist_noclobber(&path) {
        Ok(_) => Ok(output),
        Err(error) => {
            let fallback = if error.error.kind() == io::ErrorKind::AlreadyExists {
                ErrorEnvelope::destination_exists(
                    operation,
                    Some(metadata.clone()),
                    spelling.clone(),
                )
            } else {
                ErrorEnvelope::artifact_io(
                    operation,
                    Some(metadata.clone()),
                    spelling.clone(),
                    ArtifactIoReason::PublishFailed,
                )
            };
            Err(discard_error(
                error.file,
                operation,
                Some(metadata),
                &spelling,
                fallback,
            ))
        }
    }
}

fn discard_error(
    file: tempfile::NamedTempFile,
    operation: OperationContext,
    metadata: Option<ResponseMetadata>,
    spelling: &str,
    fallback: ErrorEnvelope,
) -> ErrorEnvelope {
    match file.close() {
        Ok(()) => fallback,
        Err(_) => ErrorEnvelope::artifact_io(
            operation,
            metadata,
            spelling.to_owned(),
            ArtifactIoReason::CleanupFailed,
        ),
    }
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
    reply: ArtifactReply<'_>,
    exit: u8,
) -> Result<BufferedOutput, ErrorEnvelope> {
    let envelope = BinaryResponseEnvelope {
        kind: "response",
        operation,
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
    response: BinaryResponse<'a>,
}

#[derive(Serialize)]
struct BinaryResponse<'a> {
    metadata: &'a ResponseMetadata,
    reply: ArtifactReply<'a>,
}

#[derive(Serialize)]
#[serde(tag = "kind", content = "value", rename_all = "snake_case")]
enum ArtifactReply<'a> {
    Archive(ArtifactReference<'a>),
    Status(&'a StatusEnvelope),
    Unrecognized(ArtifactReference<'a>),
}

#[derive(Serialize)]
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
