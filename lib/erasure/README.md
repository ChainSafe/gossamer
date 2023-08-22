# Building Rust code Binary and Setting Environment

- Generate rust binary
    ```
    cd lib/erasure/rustlib/ && cargo build --release --target x86_64-unknown-linux-gnu	&& cd ..
    cp ./rustlib/target/x86_64-unknown-linux-gnu/release/liberasure_coding_gorust.so ./liberasure.so && cd ../../
    ```

- set a path to .so file
    ```
    export LD_LIBRARY_PATH=${PWD}/lib/erasure
    ```