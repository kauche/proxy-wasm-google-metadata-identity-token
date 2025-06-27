ARG RUST_VERSION=1.85.1

FROM rust:${RUST_VERSION}

RUN rustup target add wasm32-wasip1 --toolchain "${RUST_VERSION}"
