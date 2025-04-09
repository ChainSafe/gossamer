/*
 * Copyright 2023 ChainSafe Systems (ON)
 * SPDX-License-Identifier: LGPL-3.0-only
 */

#include <stdlib.h>
#include <stddef.h>

const char* systematic_recovery_threshold(size_t n_validators, size_t *res_threshold);
const char* obtain_chunks(size_t n_validators, unsigned char *data, size_t len, unsigned char **flattened_chunks, size_t *flattened_chunks_len);
const char* reconstruct(size_t n_validators, unsigned char *flattened_chunks, size_t flattened_chunks_len, size_t chunk_size, unsigned char **res_data, size_t *res_len);