#!/bin/bash

#set -e

echo '>> Updating submodule...'
git submodule update --init

if ! [ -x "$(command -v rustc)" ]; then
  echo '>> rustc not found, installing...'
  curl https://sh.rustup.rs -sSf | sh -s -- -y --default-toolchain nightly
  source $HOME/.cargo/env
  rustup install nightly
  rustup default nightly
  cargo install --force cbindgen
fi

echo '>> Building from source...'
cd crypto/sr25519-crust
mkdir build
cd build
cmake .. -DCMAKE_BUILD_TYPE=Release

echo '>> Installing...'
sudo -E env "PATH=$PATH" make install