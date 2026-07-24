use std::io;
use std::net::{TcpListener, TcpStream};
use std::thread;
use std::time::{Duration, Instant};

pub(crate) fn accept_within(listener: &TcpListener) -> TcpStream {
    listener
        .set_nonblocking(true)
        .expect("configure loopback listener");
    let deadline = Instant::now() + Duration::from_secs(5);
    let (stream, _) = loop {
        match listener.accept() {
            Ok(connection) => break connection,
            Err(error) if error.kind() == io::ErrorKind::WouldBlock => {
                assert!(Instant::now() < deadline, "CLI never connected");
                thread::sleep(Duration::from_millis(5));
            }
            Err(error) => panic!("accept CLI connection: {error}"),
        }
    };
    stream
        .set_nonblocking(false)
        .expect("restore blocking fixture stream");
    stream
}
