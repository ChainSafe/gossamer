// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

extern crate reed_solomon_novelpoly;

use reed_solomon_novelpoly::{
    {CodeParams, WrappedShard},
    f2e16::FIELD_SIZE,
};
use thiserror::Error;
use std::ffi::CString;

/// Errors in erasure coding.
#[derive(Debug, Clone, PartialEq, Error)]
pub enum Error {
	/// Returned when there are too many validators.
	#[error("There are too many validators")]
	TooManyValidators,
	/// Cannot encode something for zero or one validator
	#[error("Expected at least 2 validators")]
	NotEnoughValidators,
	/// Chunks not of uniform length or the chunks are empty.
	#[error("Chunks are not uniform, mismatch in length or are zero sized")]
	NonUniformChunks,
	/// An uneven byte-length of a shard is not valid for `GF(2^16)` encoding.
	#[error("Uneven length is not valid for field GF(2^16)")]
	UnevenLength,
	/// Chunk index out of bounds.
	#[error("Chunk is out of bounds: {chunk_index} not included in 0..{n_validators}")]
	ChunkIndexOutOfBounds { chunk_index: usize, n_validators: usize },
	/// Bad payload in reconstructed bytes.
	#[error("Reconstructed payload invalid")]
	BadPayload,
	/// Unknown error
	#[error("An unknown error has appeared when deriving code parameters from validator count")]
	UnknownCodeParam,
}

fn str_to_i8ptr(error_message: String) -> *const i8 {
	let cstring = CString::new(error_message).unwrap();
    let cstring_ptr = cstring.as_ptr();
    std::mem::forget(cstring); // Prevent Rust from freeing the memory

    cstring_ptr
}

fn code_params(n_validators: usize) -> Result<CodeParams, Error> {
	// we need to be able to reconstruct from 1/3 - eps

	let n_wanted = n_validators;
	let k_wanted = recovery_threshold(n_wanted)?;

	if n_wanted > FIELD_SIZE as usize {
		return Err(Error::TooManyValidators)
	}

	CodeParams::derive_parameters(n_wanted, k_wanted).map_err(|e| match e {
		reed_solomon_novelpoly::Error::WantedShardCountTooHigh(_) => Error::TooManyValidators,
		reed_solomon_novelpoly::Error::WantedShardCountTooLow(_) => Error::NotEnoughValidators,
		_ => Error::UnknownCodeParam,
	})
}

/// Obtain a threshold of chunks that should be enough to recover the data.
pub const fn recovery_threshold(n_validators: usize) -> Result<usize, Error> {
	if n_validators > FIELD_SIZE {
		return Err(Error::TooManyValidators)
	}
	if n_validators <= 1 {
		return Err(Error::NotEnoughValidators)
	}

	let needed = n_validators.saturating_sub(1) / 3;
	Ok(needed + 1)
}

// Obtain erasure-coded chunks, one for each validator.
//
// Works only up to 65536 validators, and `n_validators` must be non-zero.
#[no_mangle]
pub extern "C" fn obtain_chunks(
	n_validators: usize, 
	data: *const u8, len: usize,
	flattened_chunks: *mut *mut u8, flattened_chunks_len: *mut usize
) -> *const i8 {

	let data_slice = unsafe { std::slice::from_raw_parts(data, len) };
	if data_slice.is_empty() {
		return str_to_i8ptr(Error::BadPayload.to_string())
	}
	
	let params_res = code_params(n_validators);
	if params_res.is_err() {
		return str_to_i8ptr(params_res.unwrap_err().to_string());
	}
	let params = params_res.unwrap();


	let shards_res = params
    .make_encoder()
    .encode::<WrappedShard>(&data_slice[..]);

	if shards_res.is_err() {
		return str_to_i8ptr(shards_res.unwrap_err().to_string());
	}

	let shards = shards_res.unwrap();
	let chunks: Vec<Vec<u8>> = shards.into_iter().map(|w: WrappedShard| w.into_inner()).collect();
	let mut flattened: Vec<u8> = chunks.iter().flat_map(|chunk| chunk.iter().cloned()).collect();

	let result_len = flattened.len();
    let result_data = flattened.as_mut_ptr();

    unsafe {
        *flattened_chunks = result_data;
        *flattened_chunks_len = result_len;
    }

	std::mem::forget(flattened);
	return std::ptr::null();
}


// Reconstruct decodable data from a set of chunks.
//
// Provide an iterator containing chunk data and the corresponding index.
// The indices of the present chunks must be indicated. If too few chunks
// are provided, recovery is not possible.
//
// Works only up to 65536 validators, and `n_validators` must be non-zero
#[no_mangle]
pub extern "C" fn reconstruct(
	n_validators: usize,
	flattened_chunks: *const u8, flattened_chunks_len: usize,
	chunk_size: usize,
	res_data: *mut *mut u8, res_len: *mut usize
) -> *const i8 {

	let flattened_slice: &[u8] = unsafe { std::slice::from_raw_parts(flattened_chunks, flattened_chunks_len) };

	let chunks = flattened_slice
		.chunks(chunk_size)
		.enumerate()
		.map(|(index, chunk)| (chunk, index));

		let params_res = code_params(n_validators);
		if params_res.is_err() {
			return str_to_i8ptr(params_res.unwrap_err().to_string());
		}
		let params = params_res.unwrap();

		let mut received_shards: Vec<Option<WrappedShard>> = vec![None; n_validators];
		let mut shard_len = None;
		for (chunk_data, chunk_idx) in chunks.into_iter().take(n_validators) {
			if chunk_idx >= n_validators {
				return str_to_i8ptr(Error::ChunkIndexOutOfBounds { chunk_index: chunk_idx, n_validators }.to_string());
			}
	
			let shard_len = shard_len.get_or_insert_with(|| chunk_data.len());
	
			if *shard_len % 2 != 0 {
				return str_to_i8ptr(Error::UnevenLength.to_string());
			}
	
			if *shard_len != chunk_data.len() || *shard_len == 0 {
				return str_to_i8ptr(Error::NonUniformChunks.to_string())
			}
	
			received_shards[chunk_idx] = Some(WrappedShard::new(chunk_data.to_vec()));
		}
	
		let reconstruct_res = params.make_encoder().reconstruct(received_shards);
		if reconstruct_res.is_err() {
			return str_to_i8ptr(reconstruct_res.unwrap_err().to_string());
		}
		let mut res_vec = reconstruct_res.unwrap();

		unsafe {
			*res_data = res_vec.as_mut_ptr();
			*res_len = res_vec.len();
		}

		std::mem::forget(res_vec);
		return std::ptr::null();
}
