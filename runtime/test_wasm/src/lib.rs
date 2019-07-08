pub use rstd;

extern {
    fn ext_print_utf8(offset: i32, len: i32);
    fn ext_get_storage_into(key_data: i32, key_len: i32, value_data: i32, value_len: i32, value_offset: i32) -> i32;
}

#[no_mangle]
pub extern fn test_ext_print_utf8() {
	let message = rstd::alloc::format!("{}", "hello world!");
	unsafe {
		ext_print_utf8(message.as_ptr() as i32, message.len() as i32);
	}
}

#[no_mangle]
pub extern fn test_ext_get_storage_into(key_data: i32, key_len: i32, value_data: i32, value_len: i32, value_offset: i32) -> i32{
   	unsafe {
   		ext_get_storage_into(key_data, key_len, value_data, value_len, value_offset)
   	}
}