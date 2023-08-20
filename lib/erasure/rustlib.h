#include <stdlib.h>
#include <stddef.h>

int32_t add(int32_t a, int32_t b);
const char* obtain_chunks(size_t n_validators, u_char *data, size_t len, u_char **flattened_chunks, size_t *flattened_chunks_len);
const char* reconstruct(size_t n_validators, u_char *flattened_chunks, size_t flattened_chunks_len, size_t chunk_size, u_char **res_data, size_t *res_len);
const char* try_error();