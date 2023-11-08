package allocator

import (
	"encoding/binary"
	"testing"
)

type MemoryInstance struct {
	data         []byte
	maxWasmPages uint32
}

//nolint:unparam
func (m *MemoryInstance) setMaxWasmPages(max uint32) {
	m.maxWasmPages = max
}

func (m *MemoryInstance) pages() uint32 {
	pages, ok := pagesFromSize(uint64(len(m.data)))
	if !ok {
		panic("cannot get page number")
	}
	return pages
}

func (m *MemoryInstance) Size() uint32 {
	return m.pages() * PageSize
}

func (m *MemoryInstance) Grow(pages uint32) (uint32, bool) {
	if m.pages()+pages > m.maxWasmPages {
		return 0, false
	}

	prevPages := m.pages()

	resizedLinearMem := make([]byte, (m.pages()+pages)*PageSize)
	copy(resizedLinearMem[0:len(m.data)], m.data)
	m.data = resizedLinearMem
	return prevPages, true
}

//nolint:govet
func (m *MemoryInstance) ReadByte(offset uint32) (byte, bool) { return 0x00, false }
func (m *MemoryInstance) ReadUint64Le(offset uint32) (uint64, bool) {
	return binary.LittleEndian.Uint64(m.data[offset : offset+8]), true
}
func (m *MemoryInstance) WriteUint64Le(offset uint32, v uint64) bool {
	encoded := make([]byte, 8)
	binary.LittleEndian.PutUint64(encoded, v)
	copy(m.data[offset:offset+8], encoded)
	return true
}
func (m *MemoryInstance) Read(offset, byteCount uint32) ([]byte, bool) {
	return nil, false
}

//nolint:govet
func (m *MemoryInstance) WriteByte(offset uint32, v byte) bool {
	return false
}
func (m *MemoryInstance) Write(offset uint32, v []byte) bool {
	return false
}

func NewMemoryInstanceWithPages(t *testing.T, pages uint32) *MemoryInstance {
	t.Helper()
	return &MemoryInstance{
		data:         make([]byte, pages*PageSize),
		maxWasmPages: MaxWasmPages,
	}
}
