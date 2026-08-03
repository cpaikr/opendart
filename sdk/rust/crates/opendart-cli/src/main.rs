//! Machine-readable OpenDART CLI backed by the first-party typed Rust SDK.
#![forbid(unsafe_code)]

mod app;
mod artifact;
mod command;
mod discovery;
mod error;
mod execution;
#[path = "generated/dispatch/mod.rs"]
#[rustfmt::skip]
mod dispatch_projection;
#[path = "generated/interface/mod.rs"]
#[rustfmt::skip]
mod interface_projection;
mod output;
mod prepared;

fn main() -> std::process::ExitCode {
    std::process::ExitCode::from(app::run(std::env::args_os()))
}
