use std::ffi::{OsStr, OsString};
use std::io::{self, Write};
use std::path::PathBuf;
use std::sync::Arc;
use std::sync::atomic::{AtomicU8, Ordering};

use cap_std::ambient_authority;
#[cfg(windows)]
use cap_std::fs::OpenOptionsExt;
use cap_std::fs::{Dir, DirBuilder, File, OpenOptions};
#[cfg(unix)]
use cap_std::fs::{DirBuilderExt, OpenOptionsExt};
use tokio::sync::mpsc;

use super::{ArtifactTarget, ArtifactWriteError, write_bounded};
use crate::error::CleanupContext;

const COMMAND_CAPACITY: usize = 1;
const STAGED_FILE_NAME: &str = "artifact";
const ACTIVE: u8 = 0;
const CANCELLED: u8 = 1;
const COMMITTING: u8 = 2;
const COMMITTED: u8 = 3;

#[derive(Clone, Copy, Debug)]
pub(super) enum FailureKind {
    DestinationExists,
    DestinationMetadataUnavailable,
    ParentNotDirectory,
    ParentUnavailable,
    TemporaryFileCreation,
    Limit,
    Write,
    Flush,
    Publish,
    WorkerUnavailable,
}

#[derive(Debug)]
pub(super) struct WorkerError {
    pub(super) kind: FailureKind,
    pub(super) cleanup: Option<CleanupContext>,
}

pub(super) struct ArtifactTransaction {
    commands: mpsc::Sender<WorkerCommand>,
    events: mpsc::UnboundedReceiver<WorkerEvent>,
    commit_authority: Arc<CommitAuthority>,
}

impl ArtifactTransaction {
    pub(super) async fn start(target: ArtifactTarget) -> Result<Self, WorkerError> {
        Self::start_with_hook(target, WorkerHook::default()).await
    }

    async fn start_with_hook(
        target: ArtifactTarget,
        hook: WorkerHook,
    ) -> Result<Self, WorkerError> {
        let spec = WorkerSpec::from(target);
        let (commands, receiver) = mpsc::channel(COMMAND_CAPACITY);
        let (events_sender, events) = mpsc::unbounded_channel();
        let commit_authority = Arc::new(CommitAuthority::new());
        let worker_authority = Arc::clone(&commit_authority);
        std::thread::Builder::new()
            .name("opendart-artifact".to_owned())
            .spawn(move || {
                run_worker(spec, receiver, events_sender, worker_authority, hook);
            })
            .map_err(|_| WorkerError {
                kind: FailureKind::WorkerUnavailable,
                cleanup: None,
            })?;
        let mut transaction = Self {
            commands,
            events,
            commit_authority,
        };
        match transaction.events.recv().await {
            Some(WorkerEvent::Ready) => Ok(transaction),
            Some(WorkerEvent::Stopped { failure, cleanup }) => Err(WorkerError {
                kind: failure.unwrap_or(FailureKind::TemporaryFileCreation),
                cleanup,
            }),
            Some(WorkerEvent::Finished(_)) | Some(WorkerEvent::Committed { .. }) | None => {
                Err(WorkerError {
                    kind: FailureKind::WorkerUnavailable,
                    cleanup: None,
                })
            }
        }
    }

    pub(super) async fn enqueue(&mut self, chunk: Vec<u8>) -> Result<(), WorkerError> {
        if self
            .commands
            .send(WorkerCommand::Write(chunk))
            .await
            .is_err()
        {
            return Err(self.stopped_error(FailureKind::Write).await);
        }
        Ok(())
    }

    pub(super) async fn finish(&mut self) -> Result<u64, WorkerError> {
        if self.commands.send(WorkerCommand::Finish).await.is_err() {
            return Err(self.stopped_error(FailureKind::Flush).await);
        }
        match self.events.recv().await {
            Some(WorkerEvent::Finished(bytes)) => Ok(bytes),
            Some(WorkerEvent::Stopped { failure, cleanup }) => Err(WorkerError {
                kind: failure.unwrap_or(FailureKind::Flush),
                cleanup,
            }),
            Some(WorkerEvent::Ready) | Some(WorkerEvent::Committed { .. }) | None => {
                Err(WorkerError {
                    kind: FailureKind::WorkerUnavailable,
                    cleanup: None,
                })
            }
        }
    }

    pub(super) async fn commit(mut self) -> Result<Option<CleanupContext>, WorkerError> {
        if self.commands.send(WorkerCommand::Commit).await.is_err() {
            return Err(self.stopped_error(FailureKind::Publish).await);
        }
        match self.events.recv().await {
            Some(WorkerEvent::Committed { cleanup }) => Ok(cleanup),
            Some(WorkerEvent::Stopped { failure, cleanup }) => Err(WorkerError {
                kind: failure.unwrap_or(FailureKind::Publish),
                cleanup,
            }),
            Some(WorkerEvent::Ready) | Some(WorkerEvent::Finished(_)) | None => Err(WorkerError {
                kind: FailureKind::WorkerUnavailable,
                cleanup: None,
            }),
        }
    }

    pub(super) async fn discard(mut self) -> Option<CleanupContext> {
        let _ = self.commit_authority.revoke();
        let _ = self.commands.send(WorkerCommand::Discard).await;
        loop {
            match self.events.recv().await {
                Some(WorkerEvent::Stopped { cleanup, .. }) => return cleanup,
                Some(WorkerEvent::Ready | WorkerEvent::Finished(_)) => {}
                Some(WorkerEvent::Committed { cleanup }) => return cleanup,
                None => return None,
            }
        }
    }

    pub(super) fn abandon(self) -> CleanupContext {
        // Sending Commit consumes the transaction, so an owner that can call
        // abandon still has exclusive authority to revoke publication.
        let revoked = self.commit_authority.revoke();
        debug_assert!(revoked, "only an active transaction can be abandoned");
        CleanupContext::staging_cleanup_pending()
    }

    async fn stopped_error(&mut self, fallback: FailureKind) -> WorkerError {
        match self.events.recv().await {
            Some(WorkerEvent::Stopped { failure, cleanup }) => WorkerError {
                kind: failure.unwrap_or(fallback),
                cleanup,
            },
            _ => WorkerError {
                kind: FailureKind::WorkerUnavailable,
                cleanup: None,
            },
        }
    }
}

impl Drop for ArtifactTransaction {
    fn drop(&mut self) {
        let _ = self.commit_authority.revoke();
    }
}

struct CommitAuthority {
    state: AtomicU8,
}

impl CommitAuthority {
    const fn new() -> Self {
        Self {
            state: AtomicU8::new(ACTIVE),
        }
    }

    fn is_active(&self) -> bool {
        self.state.load(Ordering::Acquire) == ACTIVE
    }

    fn revoke(&self) -> bool {
        self.state
            .compare_exchange(ACTIVE, CANCELLED, Ordering::AcqRel, Ordering::Acquire)
            .is_ok()
    }

    fn begin_commit(&self) -> bool {
        self.state
            .compare_exchange(ACTIVE, COMMITTING, Ordering::AcqRel, Ordering::Acquire)
            .is_ok()
    }

    fn mark_committed(&self) {
        let previous = self.state.swap(COMMITTED, Ordering::AcqRel);
        debug_assert_eq!(previous, COMMITTING);
    }
}

struct WorkerSpec {
    parent: PathBuf,
    destination: OsString,
    limit: u64,
}

impl From<ArtifactTarget> for WorkerSpec {
    fn from(target: ArtifactTarget) -> Self {
        let parent = super::artifact_parent(&target.path).to_owned();
        let destination = target
            .path
            .file_name()
            .expect("validated artifact paths have a final component")
            .to_owned();
        Self {
            parent,
            destination,
            limit: target.limit,
        }
    }
}

enum WorkerCommand {
    Write(Vec<u8>),
    Finish,
    Commit,
    Discard,
}

enum WorkerEvent {
    Ready,
    Finished(u64),
    Committed {
        cleanup: Option<CleanupContext>,
    },
    Stopped {
        failure: Option<FailureKind>,
        cleanup: Option<CleanupContext>,
    },
}

fn run_worker(
    spec: WorkerSpec,
    mut commands: mpsc::Receiver<WorkerCommand>,
    events: mpsc::UnboundedSender<WorkerEvent>,
    commit_authority: Arc<CommitAuthority>,
    hook: WorkerHook,
) {
    let mut staged = match StagedArtifact::create(spec) {
        Ok(staged) => staged,
        Err(error) => {
            let _ = events.send(WorkerEvent::Stopped {
                failure: Some(error.kind),
                cleanup: error.cleanup,
            });
            return;
        }
    };
    if events.send(WorkerEvent::Ready).is_err() {
        staged.cleanup();
        return;
    }

    let mut finished = false;
    loop {
        if !commit_authority.is_active() {
            let cleanup = staged.cleanup();
            let _ = events.send(WorkerEvent::Stopped {
                failure: None,
                cleanup,
            });
            return;
        }
        let Some(command) = commands.blocking_recv() else {
            let cleanup = staged.cleanup();
            let _ = events.send(WorkerEvent::Stopped {
                failure: None,
                cleanup,
            });
            return;
        };
        match command {
            WorkerCommand::Write(chunk) if !finished => {
                hook.before_write();
                if !commit_authority.is_active() {
                    continue;
                }
                let result = if hook.fail_write() {
                    Err(FailureKind::Write)
                } else {
                    staged.write(&chunk)
                };
                if let Err(kind) = result {
                    let cleanup = staged.cleanup();
                    let _ = events.send(WorkerEvent::Stopped {
                        failure: Some(kind),
                        cleanup,
                    });
                    return;
                }
            }
            WorkerCommand::Finish if !finished => match staged.flush() {
                Ok(bytes) => {
                    finished = true;
                    if events.send(WorkerEvent::Finished(bytes)).is_err() {
                        staged.cleanup();
                        return;
                    }
                }
                Err(kind) => {
                    let cleanup = staged.cleanup();
                    let _ = events.send(WorkerEvent::Stopped {
                        failure: Some(kind),
                        cleanup,
                    });
                    return;
                }
            },
            WorkerCommand::Commit if finished => {
                if !commit_authority.begin_commit() {
                    continue;
                }
                match staged.commit(&hook) {
                    Ok(cleanup) => {
                        commit_authority.mark_committed();
                        let _ = events.send(WorkerEvent::Committed { cleanup });
                    }
                    Err(error) => {
                        let _ = events.send(WorkerEvent::Stopped {
                            failure: Some(error.kind),
                            cleanup: error.cleanup,
                        });
                    }
                }
                return;
            }
            WorkerCommand::Discard => {
                let cleanup = staged.cleanup();
                let _ = events.send(WorkerEvent::Stopped {
                    failure: None,
                    cleanup,
                });
                return;
            }
            WorkerCommand::Write(_) | WorkerCommand::Finish | WorkerCommand::Commit => {
                let cleanup = staged.cleanup();
                let _ = events.send(WorkerEvent::Stopped {
                    failure: Some(FailureKind::WorkerUnavailable),
                    cleanup,
                });
                return;
            }
        }
    }
}

struct StagedArtifact {
    parent: Dir,
    stage: Option<Dir>,
    file: Option<File>,
    destination: OsString,
    bytes: u64,
    limit: u64,
}

impl StagedArtifact {
    fn create(spec: WorkerSpec) -> Result<Self, WorkerError> {
        let parent = Dir::open_ambient_dir(&spec.parent, ambient_authority()).map_err(|error| {
            let kind = if error.kind() == io::ErrorKind::NotADirectory {
                FailureKind::ParentNotDirectory
            } else {
                FailureKind::ParentUnavailable
            };
            WorkerError {
                kind,
                cleanup: None,
            }
        })?;
        match parent.symlink_metadata(&spec.destination) {
            Ok(_) => {
                return Err(WorkerError {
                    kind: FailureKind::DestinationExists,
                    cleanup: None,
                });
            }
            Err(error) if error.kind() == io::ErrorKind::NotFound => {}
            Err(_) => {
                return Err(WorkerError {
                    kind: FailureKind::DestinationMetadataUnavailable,
                    cleanup: None,
                });
            }
        }
        let stage = create_private_stage(&parent)?;
        let file = match create_staged_file(&stage) {
            Ok(file) => file,
            Err(_) => {
                let cleanup = cleanup_stage(stage, None);
                return Err(WorkerError {
                    kind: FailureKind::TemporaryFileCreation,
                    cleanup,
                });
            }
        };
        Ok(Self {
            parent,
            stage: Some(stage),
            file: Some(file),
            destination: spec.destination,
            bytes: 0,
            limit: spec.limit,
        })
    }

    fn write(&mut self, chunk: &[u8]) -> Result<(), FailureKind> {
        let file = self.file.as_mut().ok_or(FailureKind::WorkerUnavailable)?;
        self.bytes =
            write_bounded(file, self.bytes, chunk, self.limit).map_err(|error| match error {
                ArtifactWriteError::Limit => FailureKind::Limit,
                ArtifactWriteError::Io => FailureKind::Write,
            })?;
        Ok(())
    }

    fn flush(&mut self) -> Result<u64, FailureKind> {
        self.file
            .as_mut()
            .ok_or(FailureKind::WorkerUnavailable)?
            .flush()
            .map_err(|_| FailureKind::Flush)?;
        Ok(self.bytes)
    }

    fn commit(mut self, hook: &WorkerHook) -> Result<Option<CleanupContext>, WorkerError> {
        let stage = self.stage.as_ref().ok_or(WorkerError {
            kind: FailureKind::WorkerUnavailable,
            cleanup: None,
        })?;
        let file = self.file.as_ref().ok_or(WorkerError {
            kind: FailureKind::WorkerUnavailable,
            cleanup: None,
        })?;
        hook.before_publish();
        let result = if hook.fail_publish() {
            Err(io::Error::other(
                "compatibility-injected publication failure",
            ))
        } else {
            publish_retained(file, stage, &self.parent, &self.destination)
        };
        if let Err(error) = result {
            let kind = if error.kind() == io::ErrorKind::AlreadyExists {
                FailureKind::DestinationExists
            } else {
                FailureKind::Publish
            };
            let cleanup = self.cleanup();
            return Err(WorkerError { kind, cleanup });
        }
        Ok(self.cleanup())
    }

    fn cleanup(&mut self) -> Option<CleanupContext> {
        let file = self.file.take();
        let stage = self.stage.take();
        stage.and_then(|stage| cleanup_stage(stage, file))
    }
}

fn create_private_stage(parent: &Dir) -> Result<Dir, WorkerError> {
    #[cfg(unix)]
    let builder = {
        let mut builder = DirBuilder::new();
        builder.mode(0o700);
        builder
    };
    #[cfg(not(unix))]
    let builder = DirBuilder::new();
    loop {
        let mut random = [0_u8; 16];
        getrandom::fill(&mut random).map_err(|_| WorkerError {
            kind: FailureKind::TemporaryFileCreation,
            cleanup: None,
        })?;
        let name = format!(".opendart-{:032x}", u128::from_ne_bytes(random));
        match parent.create_dir_with(&name, &builder) {
            Ok(()) => match parent.open_dir(&name) {
                Ok(stage) => return Ok(stage),
                Err(_) => {
                    return Err(WorkerError {
                        kind: FailureKind::TemporaryFileCreation,
                        cleanup: cleanup_unopened_stage(parent, &name),
                    });
                }
            },
            Err(error) if error.kind() == io::ErrorKind::AlreadyExists => {}
            Err(_) => {
                return Err(WorkerError {
                    kind: FailureKind::TemporaryFileCreation,
                    cleanup: None,
                });
            }
        }
    }
}

fn cleanup_unopened_stage(parent: &Dir, name: &str) -> Option<CleanupContext> {
    match parent.remove_dir(name) {
        Ok(()) => None,
        Err(error) if error.kind() == io::ErrorKind::NotFound => None,
        Err(_) => Some(CleanupContext::discard_staging_link()),
    }
}

fn create_staged_file(stage: &Dir) -> io::Result<File> {
    let mut options = OpenOptions::new();
    options.read(true).write(true).create_new(true);
    #[cfg(unix)]
    options.mode(0o600);
    #[cfg(windows)]
    options.share_mode(1);
    stage.open_with(STAGED_FILE_NAME, &options)
}

#[cfg(any(target_os = "linux", target_os = "android"))]
fn publish_retained(
    file: &File,
    _stage: &Dir,
    parent: &Dir,
    destination: &OsStr,
) -> io::Result<()> {
    let proc_self_fd = rustix_linux_procfs::proc_self_fd().map_err(io::Error::from)?;
    let source = rustix::path::DecInt::from_fd(file);
    rustix::fs::linkat(
        proc_self_fd,
        source,
        parent,
        destination,
        rustix::fs::AtFlags::SYMLINK_FOLLOW,
    )
    .map_err(io::Error::from)
}

#[cfg(target_os = "macos")]
fn publish_retained(
    file: &File,
    _stage: &Dir,
    parent: &Dir,
    destination: &OsStr,
) -> io::Result<()> {
    rustix::fs::fclonefileat(file, parent, destination, rustix::fs::CloneFlags::empty())
        .map_err(io::Error::from)
}

#[cfg(windows)]
fn publish_retained(
    _file: &File,
    stage: &Dir,
    parent: &Dir,
    destination: &OsStr,
) -> io::Result<()> {
    // The staged file denies deletion and write sharing while this
    // pathname-based link is created.
    stage.hard_link(STAGED_FILE_NAME, parent, destination)
}

#[cfg(not(any(
    target_os = "linux",
    target_os = "android",
    target_os = "macos",
    windows
)))]
fn publish_retained(
    _file: &File,
    _stage: &Dir,
    _parent: &Dir,
    _destination: &OsStr,
) -> io::Result<()> {
    Err(io::Error::new(
        io::ErrorKind::Unsupported,
        "identity-based artifact publication is unsupported",
    ))
}

fn cleanup_stage(stage: Dir, file: Option<File>) -> Option<CleanupContext> {
    drop(file);
    #[cfg(opendart_compat)]
    if let Some(delay) = std::env::var("OPENDART_COMPAT_ARTIFACT_CLEANUP_DELAY_MS")
        .ok()
        .and_then(|value| value.parse::<u64>().ok())
    {
        std::thread::sleep(std::time::Duration::from_millis(delay));
    }
    #[cfg(opendart_compat)]
    if std::env::var_os("OPENDART_COMPAT_ARTIFACT_CLEANUP_FAILURE").is_some() {
        drop(stage);
        return Some(CleanupContext::discard_staging_link());
    }
    stage
        .remove_open_dir_all()
        .err()
        .map(|_| CleanupContext::discard_staging_link())
}

#[derive(Default)]
struct WorkerHook {
    #[cfg(test)]
    write_stall: Option<Arc<TestStall>>,
    #[cfg(test)]
    publish_stall: Option<Arc<TestStall>>,
}

impl WorkerHook {
    fn before_write(&self) {
        #[cfg(opendart_compat)]
        if let Some(delay) = std::env::var("OPENDART_COMPAT_ARTIFACT_WRITE_DELAY_MS")
            .ok()
            .and_then(|value| value.parse::<u64>().ok())
        {
            std::thread::sleep(std::time::Duration::from_millis(delay));
        }
        #[cfg(test)]
        if let Some(stall) = &self.write_stall {
            stall.wait();
        }
    }

    fn before_publish(&self) {
        #[cfg(test)]
        if let Some(stall) = &self.publish_stall {
            stall.wait();
        }
    }

    fn fail_write(&self) -> bool {
        #[cfg(opendart_compat)]
        let failure = std::env::var_os("OPENDART_COMPAT_ARTIFACT_WRITE_FAILURE").is_some();
        #[cfg(not(opendart_compat))]
        let failure = false;
        failure
    }

    fn fail_publish(&self) -> bool {
        #[cfg(opendart_compat)]
        let failure = std::env::var_os("OPENDART_COMPAT_ARTIFACT_PUBLISH_FAILURE").is_some();
        #[cfg(not(opendart_compat))]
        let failure = false;
        failure
    }
}

#[cfg(test)]
struct TestStall {
    entered: mpsc::UnboundedSender<()>,
    release: Arc<(std::sync::Mutex<bool>, std::sync::Condvar)>,
}

#[cfg(test)]
impl TestStall {
    fn wait(&self) {
        let _ = self.entered.send(());
        let (lock, ready) = &*self.release;
        let mut released = lock.lock().expect("stall lock");
        while !*released {
            released = ready.wait(released).expect("stall wait");
        }
    }
}

#[cfg(test)]
mod tests {
    use std::sync::{Arc, Condvar, Mutex};
    use std::time::{Duration, Instant};

    use cap_std::ambient_authority;
    use cap_std::fs::Dir;

    use super::{ArtifactTransaction, TestStall, WorkerHook, cleanup_unopened_stage};
    #[cfg(any(target_os = "linux", target_os = "macos"))]
    use super::{FailureKind, STAGED_FILE_NAME};
    use crate::artifact::ArtifactTarget;
    use crate::artifact::before_deadline;

    #[test]
    fn bounded_worker_keeps_the_current_thread_runtime_responsive_when_a_write_stalls() {
        let directory = tempfile::tempdir().unwrap();
        let destination = directory.path().join("result.zip");
        let target = ArtifactTarget {
            path: destination.clone(),
            spelling: destination.to_string_lossy().into_owned(),
            limit: 64,
        };
        let (entered, mut entered_receiver) = tokio::sync::mpsc::unbounded_channel();
        let release = Arc::new((Mutex::new(false), Condvar::new()));
        let hook = WorkerHook {
            write_stall: Some(Arc::new(TestStall {
                entered,
                release: Arc::clone(&release),
            })),
            publish_stall: None,
        };
        let runtime = tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .build()
            .unwrap();

        runtime.block_on(async {
            let mut transaction = ArtifactTransaction::start_with_hook(target, hook)
                .await
                .unwrap();
            transaction.enqueue(b"first".to_vec()).await.unwrap();
            entered_receiver.recv().await.unwrap();
            transaction.enqueue(b"second".to_vec()).await.unwrap();

            let blocked = before_deadline(
                tokio::time::Instant::now() + Duration::from_millis(20),
                transaction.enqueue(b"third".to_vec()),
            )
            .await;
            assert!(
                blocked.is_err(),
                "the bounded channel must apply backpressure"
            );
            let _cleanup = transaction.abandon();
        });

        assert!(!destination.exists());
        let (lock, ready) = &*release;
        *lock.lock().unwrap() = true;
        ready.notify_all();

        let cleanup_deadline = Instant::now() + Duration::from_secs(2);
        while std::fs::read_dir(directory.path()).unwrap().count() != 0 {
            assert!(
                Instant::now() < cleanup_deadline,
                "the detached worker did not finish deferred cleanup"
            );
            std::thread::sleep(Duration::from_millis(10));
        }
    }

    #[test]
    fn commit_handoff_cannot_be_revoked_after_publication_begins() {
        let directory = tempfile::tempdir().unwrap();
        let destination = directory.path().join("result.zip");
        let target = ArtifactTarget {
            path: destination.clone(),
            spelling: destination.to_string_lossy().into_owned(),
            limit: 64,
        };
        let (entered, mut entered_receiver) = tokio::sync::mpsc::unbounded_channel();
        let release = Arc::new((Mutex::new(false), Condvar::new()));
        let hook = WorkerHook {
            write_stall: None,
            publish_stall: Some(Arc::new(TestStall {
                entered,
                release: Arc::clone(&release),
            })),
        };
        let runtime = tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .build()
            .unwrap();

        runtime.block_on(async {
            let mut transaction = ArtifactTransaction::start_with_hook(target, hook)
                .await
                .unwrap();
            transaction.enqueue(b"original".to_vec()).await.unwrap();
            assert_eq!(transaction.finish().await.unwrap(), 8);
            let authority = Arc::clone(&transaction.commit_authority);
            let commit = tokio::spawn(transaction.commit());

            entered_receiver.recv().await.unwrap();
            assert!(
                !authority.revoke(),
                "commit must own the outcome after the acknowledged handoff"
            );
            let (lock, ready) = &*release;
            *lock.lock().unwrap() = true;
            ready.notify_all();
            assert!(commit.await.unwrap().unwrap().is_none());
        });

        assert_eq!(std::fs::read(destination).unwrap(), b"original");
    }

    #[cfg(any(target_os = "linux", target_os = "macos"))]
    #[test]
    fn publication_never_uses_a_replaced_staging_path() {
        let directory = tempfile::tempdir().unwrap();
        let destination = directory.path().join("result.zip");
        let target = ArtifactTarget {
            path: destination.clone(),
            spelling: destination.to_string_lossy().into_owned(),
            limit: 64,
        };
        let (entered, mut entered_receiver) = tokio::sync::mpsc::unbounded_channel();
        let release = Arc::new((Mutex::new(false), Condvar::new()));
        let hook = WorkerHook {
            write_stall: None,
            publish_stall: Some(Arc::new(TestStall {
                entered,
                release: Arc::clone(&release),
            })),
        };
        let runtime = tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .build()
            .unwrap();

        runtime.block_on(async {
            let mut transaction = ArtifactTransaction::start_with_hook(target, hook)
                .await
                .unwrap();
            transaction.enqueue(b"original".to_vec()).await.unwrap();
            assert_eq!(transaction.finish().await.unwrap(), 8);
            let commit = tokio::spawn(transaction.commit());

            entered_receiver.recv().await.unwrap();
            let stage = std::fs::read_dir(directory.path())
                .unwrap()
                .next()
                .unwrap()
                .unwrap()
                .path();
            let staged_file = stage.join(STAGED_FILE_NAME);
            std::fs::remove_file(&staged_file).unwrap();
            std::fs::write(&staged_file, b"replacement").unwrap();

            let (lock, ready) = &*release;
            *lock.lock().unwrap() = true;
            ready.notify_all();
            match commit.await.unwrap() {
                Ok(cleanup) => {
                    assert!(cleanup.is_none());
                    assert_eq!(std::fs::read(&destination).unwrap(), b"original");
                }
                Err(error) => {
                    assert!(matches!(error.kind, FailureKind::Publish));
                    assert!(error.cleanup.is_none());
                    assert!(!destination.exists());
                }
            }
        });
    }

    #[test]
    fn unopened_stage_cleanup_failure_is_reported() {
        let directory = tempfile::tempdir().unwrap();
        let parent = Dir::open_ambient_dir(directory.path(), ambient_authority()).unwrap();
        parent.create_dir("stage").unwrap();
        std::fs::write(directory.path().join("stage").join("keep"), b"keep").unwrap();

        assert!(cleanup_unopened_stage(&parent, "stage").is_some());
    }
}
