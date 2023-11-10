package allocator

import (
	"errors"
	"fmt"
	"math"
	"math/bits"

	"github.com/tetratelabs/wazero/api"
)

const (
	Aligment = 8

	// each pointer is prefixed with 8 bytes, wich indentifies the list
	// index to which it belongs
	HeaderSize = 8

	// The minimum possible allocation size is choosen to be 8 bytes
	// because in that case we would have easier time to provide the
	// guaranteed alignment of 8
	//
	// The maximum possible allocation size is set to 32Mib
	//
	// NumOrders represents the number of orders supported, this number
	// corresponds to the number of powers between the minimum an maximum
	// possible allocation (2^3 ... 2^25 both ends inclusive)
	NumOrders              = 23
	MinPossibleAllocations = 8
	MaxPossibleAllocations = (1 << 25)

	PageSize     = 65536
	MaxWasmPages = 4 * 1024 * 1024 * 1024 / PageSize
)

var (
	ErrInvalidOrder                 = errors.New("invalid order")
	ErrRequestedAllocationTooLarge  = errors.New("requested allocation too large")
	ErrCannotReadHeader             = errors.New("cannot read header")
	ErrCannotWriteHeader            = errors.New("cannot write header")
	ErrInvalidHeaderPointerDetected = errors.New("invalid header pointer detected")
	ErrAllocatorOutOfSpace          = errors.New("allocator out of space")
	ErrCannotGrowLinearMemory       = errors.New("cannot grow linear memory")
	ErrInvalidPointerForDealocation = errors.New("invalid pointer for deallocation")
	ErrEmptyHeader                  = errors.New("allocation points to an empty header")
)

// The exponent for the power of two sized block adjusted to the minimum size.
//
// This way, if `MIN_POSSIBLE_ALLOCATION == 8`, we would get:
//
// power_of_two_size | order
// 8                 | 0
// 16                | 1
// 32                | 2
// 64                | 3
// ...
// 16777216          | 21
// 33554432          | 22
//
// and so on.
type Order uint32

func (order Order) size() uint32 {
	return MinPossibleAllocations << order
}

func (order Order) intoRaw() uint32 {
	return uint32(order)
}

func orderFromRaw(order uint32) (Order, error) {
	if order < NumOrders {
		return Order(order), nil
	}

	return Order(0), fmt.Errorf("%w: order %d should be less than %d",
		ErrInvalidOrder, order, NumOrders)
}

func orderFromSize(size uint32) (Order, error) {
	if size > MaxPossibleAllocations {
		return Order(0), fmt.Errorf("%w, requested %d, max possible allocations: %d",
			ErrRequestedAllocationTooLarge, size, MaxPossibleAllocations)
	}

	if size < MinPossibleAllocations {
		size = MinPossibleAllocations
	}

	// Round the clamped size to the next power of two.
	// It returns the unchanged value if the value is already a power of two.
	powerOfTwoSize := nextPowerOf2GT8(size)

	// Compute the number of trailing zeroes to get the order. We adjust it by the number of
	// trailing zeroes in the minimum possible allocation.
	value := bits.TrailingZeros32(powerOfTwoSize) - bits.TrailingZeros32(MinPossibleAllocations)
	return Order(value), nil
}

// A special magic value for a pointer in a link that denotes the end of the linked list.
const NilMarker = math.MaxUint32

// A link between headers in the free list.
type Link interface {
	isLink()
	intoRaw() uint32
}

// Nil, denotes that there is no next element.
type Nil struct{}

func (Nil) isLink() {}
func (Nil) intoRaw() uint32 {
	return NilMarker
}

// Link to the next element represented as a pointer to the a header.
type Ptr struct {
	headerPtr uint32
}

func (Ptr) isLink() {}
func (p Ptr) intoRaw() uint32 {
	return p.headerPtr
}

var _ Link = (*Nil)(nil)
var _ Link = (*Ptr)(nil)

func linkFromRaw(raw uint32) Link {
	if raw != NilMarker {
		return Ptr{headerPtr: raw}
	}
	return Nil{}
}

// A header of an allocation.
//
// The header is encoded in memory as follows.
//
// ## Free header
//
// ```ignore
// 64             32                  0
//
//	+--------------+-------------------+
//
// |            0 | next element link |
// +--------------+-------------------+
// ```
// ## Occupied header
// ```ignore
// 64             32                  0
//
//	+--------------+-------------------+
//
// |            1 |             order |
// +--------------+-------------------+
// ```
type Header interface {
	isHeader()
	intoOccupied() (Order, bool)
	intoFree() (Link, bool)
}

// A free header contains a link to the next element to form a free linked list.
type Free struct {
	link Link
}

func (Free) isHeader() {}
func (f Free) intoOccupied() (Order, bool) {
	return Order(0), false
}
func (f Free) intoFree() (Link, bool) {
	return f.link, true
}

// An occupied header has an attached order to know in which free list we should
// put the allocation upon deallocation
type Occupied struct {
	order Order
}

func (Occupied) isHeader() {}
func (f Occupied) intoOccupied() (Order, bool) {
	return f.order, true
}
func (f Occupied) intoFree() (Link, bool) {
	return nil, false
}

var _ Header = (*Free)(nil)
var _ Header = (*Occupied)(nil)

// readHeaderFromMemory reads a header from memory, returns an error if ther
// headerPtr is out of bounds of the linear memory or if the read header is
// corrupted (e.g the order is incorrect)
func readHeaderFromMemory(mem api.Memory, headerPtr uint32) (Header, error) {
	rawHeader, ok := mem.ReadUint64Le(headerPtr)
	if !ok {
		return nil, fmt.Errorf("%w: pointer: %d", ErrCannotReadHeader, headerPtr)
	}

	// check if the header represents an occupied or free allocation
	// and extract the header data by timing (and discarding) the high bits
	occupied := rawHeader&0x00000001_00000000 != 0
	headerData := uint32(rawHeader)

	if occupied {
		order, err := orderFromRaw(headerData)
		if err != nil {
			return nil, fmt.Errorf("order from raw: %w", err)
		}
		return Occupied{order}, nil
	}

	return Free{link: linkFromRaw(headerData)}, nil
}

// writeHeaderInto write out this header to memory, returns an error if the
// `header_ptr` is out of bounds of the linear memory.
func writeHeaderInto(header Header, mem api.Memory, headerPtr uint32) error {
	var (
		headerData   uint64
		occupiedMask uint64
	)

	switch v := header.(type) {
	case Occupied:
		headerData = uint64(v.order.intoRaw())
		occupiedMask = 0x00000001_00000000
	case Free:
		headerData = uint64(v.link.intoRaw())
		occupiedMask = 0x00000000_00000000
	default:
		panic(fmt.Sprintf("header type %T not supported", header))
	}

	rawHeader := headerData | occupiedMask
	ok := mem.WriteUint64Le(headerPtr, rawHeader)
	if !ok {
		return fmt.Errorf("%w: pointer: %d", ErrCannotWriteHeader, headerPtr)
	}
	return nil
}

// This struct represents a collection of intrusive linked lists for each order.
type FreeLists struct {
	heads [NumOrders]Link
}

func NewFreeLists() *FreeLists {
	// initialize all entries with Nil{}
	// same as [Link::Nil; N_ORDERS]
	free := [NumOrders]Link{}
	for idx := 0; idx < NumOrders; idx++ {
		free[idx] = Nil{}
	}

	return &FreeLists{
		heads: free,
	}
}

// replace replaces a given link for the specified order and returns the old one
func (f *FreeLists) replace(order Order, new Link) (old Link) {
	prev := f.heads[order]
	f.heads[order] = new
	return prev
}

type FreeingBumpHeapAllocator struct {
	originalHeapBase uint32
	bumper           uint32
	freeLists        *FreeLists
}

func NewFreeingBumpHeapAllocator(heapBase uint32) *FreeingBumpHeapAllocator {
	alignedHeapBase := (heapBase + Aligment - 1) / Aligment * Aligment
	return &FreeingBumpHeapAllocator{
		originalHeapBase: alignedHeapBase,
		bumper:           alignedHeapBase,
		freeLists:        NewFreeLists(),
	}
}

// Allocate gets the requested number of bytes to allocate and returns a pointer.
// The maximum size which can be allocated is 32MiB.
// There is no minimum size, but whatever size is passed into this function is rounded
// to the next power of two. If the requested size is bellow 8 bytes it will be rounded
// up to 8 bytes.
//
// The identity or the type of the passed memory object does not matter. However, the size
// of memory cannot shrink compared to the memory passed in previous invocations.
//
// NOTE: Once the allocator has returned an error all subsequent requests will return an error.
//
// - size: size in bytes of the allocation request
func (f *FreeingBumpHeapAllocator) Allocate(mem api.Memory, size uint32) (uint32, error) {
	fmt.Printf(">> ALLOCATE, mem size: %d, mem pages: %d\n", mem.Size(), mem.Size()/PageSize)
	// TODO: check for poisoning, implement poison bomb also observe_memory_size function
	order, err := orderFromSize(size)
	if err != nil {
		return 0, fmt.Errorf("order from size: %w", err)
	}

	var headerPtr uint32

	link := f.freeLists.heads[order]
	switch value := link.(type) {
	case Ptr:
		if uint64(value.headerPtr)+uint64(order.size())+uint64(HeaderSize) > uint64(mem.Size()) {
			return 0, fmt.Errorf("%w: pointer: %d, order size: %d",
				ErrInvalidHeaderPointerDetected, value.headerPtr, order.size())
		}

		// Remove this header from the free list.
		header, err := readHeaderFromMemory(mem, value.headerPtr)
		if err != nil {
			return 0, fmt.Errorf("reading header from memory: %w", err)
		}

		nextFree, ok := header.intoFree()
		if !ok {
			return 0, errors.New("free list points to a occupied header")
		}

		f.freeLists.heads[order] = nextFree
		headerPtr = value.headerPtr
	case Nil:
		// Corresponding free list is empty. Allocate a new item
		newPtr, err := bump(&f.bumper, order.size()+HeaderSize, mem)
		if err != nil {
			return 0, fmt.Errorf("bumping: %w", err)
		}
		headerPtr = newPtr
	default:
		panic(fmt.Sprintf("link type %T not supported", link))
	}

	// Write the order in the occupied header
	err = writeHeaderInto(Occupied{order}, mem, headerPtr)
	if err != nil {
		return 0, fmt.Errorf("writing header into: %w", err)
	}

	// TODO: allocation stats update, and bomb disarm
	return headerPtr + HeaderSize, nil
}

func (f *FreeingBumpHeapAllocator) Deallocate(mem api.Memory, ptr uint32) error {
	// TODO: check for poison, also start poinson bomb

	headerPtr, ok := checkedSub(ptr, HeaderSize)
	if !ok {
		return fmt.Errorf("%w: %d", ErrInvalidPointerForDealocation, ptr)
	}

	header, err := readHeaderFromMemory(mem, headerPtr)
	if err != nil {
		return fmt.Errorf("read header from memory: %w", err)
	}

	order, ok := header.intoOccupied()
	if !ok {
		return ErrEmptyHeader
	}

	// update the just freed header and knit it back to the free list
	prevHeader := f.freeLists.replace(order, Ptr{headerPtr})
	err = writeHeaderInto(Free{prevHeader}, mem, headerPtr)
	if err != nil {
		return fmt.Errorf("writing header into: %w", err)
	}

	//TODO: update/print stats and disarm bomb
	return nil
}

func (f *FreeingBumpHeapAllocator) Clear() {
	if f == nil {
		panic("clear cannot perform over a nil allocator")
	}

	*f = FreeingBumpHeapAllocator{
		originalHeapBase: f.originalHeapBase,
		bumper:           f.originalHeapBase,
		freeLists:        NewFreeLists(),
	}
}

func bump(bumper *uint32, size uint32, mem api.Memory) (uint32, error) {
	requiredSize := uint64(*bumper) + uint64(size)

	if requiredSize > uint64(mem.Size()) {
		requiredPages, ok := pagesFromSize(requiredSize)
		if !ok {
			return 0, fmt.Errorf("%w: required size %d dont fit uint32",
				ErrAllocatorOutOfSpace, requiredSize)
		}

		currentPages := mem.Size() / PageSize
		if currentPages >= requiredPages {
			panic(fmt.Sprintf("current pages %d >= required pages %d", currentPages, requiredPages))
		}

		if currentPages >= MaxWasmPages {
			return 0, fmt.Errorf("%w: current pages %d greater than max wasm pages %d",
				ErrAllocatorOutOfSpace, currentPages, MaxWasmPages)
		}

		if requiredPages > MaxWasmPages {
			return 0, fmt.Errorf("%w: required pages %d greater than max wasm pages %d",
				ErrAllocatorOutOfSpace, requiredPages, MaxWasmPages)
		}

		// ideally we want to double our current number of pages,
		// as long as it's less than the double absolute max we can have
		nextPages := min(currentPages*2, MaxWasmPages)
		// ... but if even more pages are required then try to allocate that many
		nextPages = max(nextPages, requiredPages)

		fmt.Printf("calling mem.Grow(%d)\n", nextPages-currentPages)
		prev, ok := mem.Grow(nextPages - currentPages)
		if !ok {
			return 0, fmt.Errorf("%w: from %d pages to %d pages",
				ErrCannotGrowLinearMemory, currentPages, nextPages)
		}

		fmt.Printf("prev pages: %d, current pages: %d\n", prev, mem.Size()/PageSize)

		pagesIncrease := (mem.Size() / PageSize) == nextPages
		if !pagesIncrease {
			panic(fmt.Sprintf("number of pages should have increased! previous: %d, desired: %d", currentPages, nextPages))
		}
	}

	res := *bumper
	*bumper += size
	return res, nil
}

// pagesFromSize convert the given `size` in bytes into the number of pages.
// The returned number of pages is ensured to be big enough to hold memory
// with the given `size`.
// Returns false if the number of pages do not fit into `u32`
func pagesFromSize(size uint64) (uint32, bool) {
	value := (size + uint64(PageSize) - 1) / uint64(PageSize)

	if value > uint64(math.MaxUint32) {
		return 0, false
	}

	return uint32(value), true
}

func checkedSub(a, b uint32) (uint32, bool) {
	if a < b {
		return 0, false
	}

	return a - b, true
}

func nextPowerOf2GT8(v uint32) uint32 {
	if v < 8 {
		return 8
	}
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}
