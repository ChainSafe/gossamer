# Generate rust binary

```
export LD_LIBRARY_PATH=${PWD}/lib/erasure
cd lib/erasure/rustlib/ && cargo build --release --target x86_64-unknown-linux-gnu && cd ..
cp ./rustlib/target/x86_64-unknown-linux-gnu/release/liberasure_coding_gorust.so liberasure_coding_gorust.so && cd ../../
```