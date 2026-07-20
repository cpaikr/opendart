//! Machine-readable OpenDART CLI backed by the first-party typed Rust SDK.
#![forbid(unsafe_code)]

mod app;
mod command;
mod discovery;
mod error;
mod execution;
#[rustfmt::skip]
mod generated;
mod output;
mod prepared;

fn main() -> std::process::ExitCode {
    std::process::ExitCode::from(app::run(std::env::args_os()))
}
