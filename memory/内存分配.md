# 内存相关概念
## 内存分页管理
### 什么是页？
将虚拟内存空间和物理内存空间按照某种规定的大小进行分配成为页(Page)
### 为什么要内存分页？
分页管理将虚拟内存空间和物理内存空间皆划分为大小相同的页面，并以页面作为内存空间的最小分配单位，一个程序的一个页面可以存放在任意一个物理页面里。
#### 内存空间碎片化
每次程序按照页进行内存分配，也就克服了外部碎片的问题。
#### 解决程序大小受限
程序将运行时所需要的页加载进内存，暂时不需要的页放在磁盘上，如果程序需要更多空间，只需再分配一个页即可

### 虚拟地址
#### 虚拟地址和物理地址

##### 物理地址
>放在寻址总线上的地址。放在寻址总线上，如果是读，电路根据这个地址每位的值就将相应地址的物理内存中的数据放到数据总线中传输。如果是写，电路根据这个地址每位的值就在相应地址的物理内存中放入数据总线上的内容。物理内存是以字节(8位)为单位编址的 <i>--摘自百度百科</i>

##### 虚拟地址
> 程序运行在虚拟地址空间中，如果CPU寄存器中的分页标志位被设置，那么执行内存操作的机器指令时MMU（内存管理单元）自动根据页目录和页表中的信息，把虚拟地址转换成物理地址，完成该指令
比如 mov eax,[004227b8h] ，这是把地址004227b8h处的值赋给寄存器的汇编代码，004227b8这个地址就是虚拟址。CPU在执行这行代码时，发现寄存器中的分页标志位已经被设定，就自动完成虚拟地址到物理地址的转换，使用物理地址取出值，完成指令。    <i>--摘自百度百科</i>

# GO内存分配
分配器将内存快分为两种
+ span:由多个连续的页组成的大内存块
+ object:将span按照特定大小切分多个小块

分配器按照按页数区分span，以页数为单位存放在管理数组中，需要时按照索引查找，获取span时如果没有合适大小则返回页数更多的span，然后进行裁剪操作，多余的部分将构成新的span放在管理数组中，分配器还尝试将地址相邻的span合并，以构建更大内存

go的内存分配操作的代码放在runtime/malloc.go文件中

分配器的数据结构为
+ fixalloc: 分配器为了方便管理和存储，将固定的非堆存储的object将在可利用空间表(free-list)
+ mspan: 运行由mheap管理的页
+ mheap: 内存分配堆，管理页的粒度（8192字节），用于管理闲置的span，需要时向操作系统申请新内存
+ mcache: 每个 Gorontine 的运行都是绑定到一个 P 上面,mcache 是每个 P 的 cache
+ mcentral: 全局 cache，mcache 不够用的时候向 mcentral 申请
+ mstats:分配统计


## fixalloc结构
fixalloc用于分配固定大小的object，Malloc使用fixalloc进行封装，管理自己的MCache和MSpan objects

fixalloc.alloc的方法返回的内存默认为0，但是调用者可以通过设置标志位为false，如果内存永远不包含堆指针这将是安全的

```go
type fixalloc struct {
	size   uintptr
	first  func(arg, p unsafe.Pointer) // called first time p is returned
	arg    unsafe.Pointer
	list   *mlink
	chunk  uintptr // use uintptr instead of unsafe.Pointer to avoid write barriers
	nchunk uint32
	inuse  uintptr // in-use bytes now
	stat   *uint64
	zero   bool // zero allocations
}
```

## mcache结构

```go
type mcache struct {
	// The following members are accessed on every malloc,
	// so they are grouped here for better caching.
	next_sample int32   // trigger heap sample after allocating this many bytes
	local_scan  uintptr // bytes of scannable heap allocated

	// Allocator cache for tiny objects w/o pointers.
	// See "Tiny allocator" comment in malloc.go.

	// tiny points to the beginning of the current tiny block, or
	// nil if there is no current tiny block.
	//
	// tiny is a heap pointer. Since mcache is in non-GC'd memory,
	// we handle it by clearing it in releaseAll during mark
	// termination.
	tiny             uintptr //小对象分配器，小于16byte的对象将会通过tiny分配
	tinyoffset       uintptr
	local_tinyallocs uintptr // number of tiny allocs not counted in other stats

	// The rest is not accessed on every malloc.

	alloc [numSpanClasses]*mspan // spans to allocate from, indexed by spanClass

	stackcache [_NumStackOrders]stackfreelist

	// Local allocator stats, flushed during GC.
	local_largefree  uintptr                  // bytes freed for large objects (>maxsmallsize)
	local_nlargefree uintptr                  // number of frees for large objects (>maxsmallsize)
	local_nsmallfree [_NumSizeClasses]uintptr // number of frees for small objects (<=maxsmallsize)
}
```

alloc [numSpanClasses]*mspan是一个大小为67的指针，指向mspan的数组，每个数组元素用来包含特定大小的块。当要分配内存大小时，为 object 在 alloc 数组中选择合适的元素来分配

相关定义在runtime/sizeclasses.go
```go
var class_to_size = [_NumSizeClasses]uint16{0, 8, 16, 32, 48, 64, 80, 96, 112, 128, 144, 160, 176, 192, 208, 224, 240, 256, 288, 320, 352, 384, 416, 448, 480, 512, 576, 640, 704, 768, 896, 1024, 1152, 1280, 1408, 1536, 1792, 2048, 2304, 2688, 3072, 3200, 3456, 4096, 4864, 5376, 6144, 6528, 6784, 6912, 8192, 9472, 9728, 10240, 10880, 12288, 13568, 14336, 16384, 18432, 19072, 20480, 21760, 24576, 27264, 28672, 32768}
```

## mspan结构
相关定义在runtime/mheap.go
```go
type mspan struct {
    // 指针域，mspan 一般都是以链表形式使用
	next *mspan     // next span in list, or nil if none
	prev *mspan     // previous span in list, or nil if none
	list *mSpanList // For debugging. TODO: Remove.

	startAddr uintptr // address of first byte of span aka s.base()
	npages    uintptr // mspan 的大小为 page 大小的整数倍

	manualFreeList gclinkptr // list of free objects in _MSpanManual spans

	// freeindex is the slot index between 0 and nelems at which to begin scanning
	// for the next free object in this span.
	// Each allocation scans allocBits starting at freeindex until it encounters a 0
	// indicating a free object. freeindex is then adjusted so that subsequent scans begin
	// just past the newly discovered free object.
	//
	// If freeindex == nelem, this span has no free objects.
	//
	// allocBits is a bitmap of objects in this span.
	// If n >= freeindex and allocBits[n/8] & (1<<(n%8)) is 0
	// then object n is free;
	// otherwise, object n is allocated. Bits starting at nelem are
	// undefined and should never be referenced.
	//
	// Object n starts at address n*elemsize + (start << pageShift).
	freeindex uintptr
	// TODO: Look up nelems from sizeclass and remove this field if it
	// helps performance.
	nelems uintptr // number of object in the span.

	// Cache of the allocBits at freeindex. allocCache is shifted
	// such that the lowest bit corresponds to the bit freeindex.
	// allocCache holds the complement of allocBits, thus allowing
	// ctz (count trailing zero) to use it directly.
	// allocCache may contain bits beyond s.nelems; the caller must ignore
	// these.
	allocCache uint64

	// allocBits and gcmarkBits hold pointers to a span's mark and
	// allocation bits. The pointers are 8 byte aligned.
	// There are three arenas where this data is held.
	// free: Dirty arenas that are no longer accessed
	//       and can be reused.
	// next: Holds information to be used in the next GC cycle.
	// current: Information being used during this GC cycle.
	// previous: Information being used during the last GC cycle.
	// A new GC cycle starts with the call to finishsweep_m.
	// finishsweep_m moves the previous arena to the free arena,
	// the current arena to the previous arena, and
	// the next arena to the current arena.
	// The next arena is populated as the spans request
	// memory to hold gcmarkBits for the next GC cycle as well
	// as allocBits for newly allocated spans.
	//
	// The pointer arithmetic is done "by hand" instead of using
	// arrays to avoid bounds checks along critical performance
	// paths.
	// The sweep will free the old allocBits and set allocBits to the
	// gcmarkBits. The gcmarkBits are replaced with a fresh zeroed
	// out memory.
	allocBits  *gcBits
	gcmarkBits *gcBits

	// sweep generation:
	// if sweepgen == h->sweepgen - 2, the span needs sweeping
	// if sweepgen == h->sweepgen - 1, the span is currently being swept
	// if sweepgen == h->sweepgen, the span is swept and ready to use
	// h->sweepgen is incremented by 2 after every GC

	sweepgen    uint32
	divMul      uint16     // for divide by elemsize - divMagic.mul
	baseMask    uint16     // if non-0, elemsize is a power of 2, & this will get object allocation base
	allocCount  uint16     // number of allocated objects
	spanclass   spanClass  // size class and noscan (uint8)
	incache     bool       // being used by an mcache
	state       mSpanState // mspaninuse etc
	needzero    uint8      // needs to be zeroed before allocation
	divShift    uint8      // for divide by elemsize - divMagic.shift
	divShift2   uint8      // for divide by elemsize - divMagic.shift2
	elemsize    uintptr    // computed from sizeclass or from npages
	unusedsince int64      // first time spotted by gc in mspanfree state
	npreleased  uintptr    // number of pages released to the os
	limit       uintptr    // end of data in span
	speciallock mutex      // guards specials list
	specials    *special   // linked list of special records sorted by offset.
}

```


+ next, prev: 指针域，mspan 一般都是以链表形式使用。
+ npages: mspan 的大小为 page 大小的整数倍。
+ spanclass: 0 ~ _NumSizeClasses 之间的一个值,比如，sizeclass = 3，那么这个 mspan 被分割成 32 byte 的块。
+ nelems: span 中包块的总数目。
+ freeindex: 0 ~ nelemes-1，表示分配到第几个块。
+ elemsize: 通过 sizeclass 或者 npages 可以计算出来。比如 sizeclass = 3, elemsize = 32 byte。对于大于 32Kb 的内存分配，都是分配整数页，elemsize = page_size * npages。

对于存储对象的object，按照8字节的倍数分为n种，例如一个24大小的object可以存储范围在17-24字节的对象

分配器初始化时会构建对照表，存储大小和规格的映射，如果对象超过特定的阀值，会被当作大对象对待

```go
_MaxSmallSize = 32 << 10 //32kb
```



## mcentral结构

```go
type mcentral struct {
	lock      mutex
	spanclass spanClass
	nonempty  mSpanList // mspan 的双向链表，当前 mcentral 中可用的 mspan list
	empty     mSpanList // 已经被使用的

	// nmalloc is the cumulative count of objects allocated from
	// this mcentral, assuming all spans in mcaches are
	// fully-allocated. Written atomically, read under STW.
	nmalloc uint64
}
```


## mheap结构

```go
type mheap struct {
	lock      mutex
	free      [_MaxMHeapList]mSpanList // 页数在_MaxMHeapList(127)以内的闲置span链表
	freelarge mTreap                   // 页数 >= _MaxMHeapList闲置的span链表
	busy      [_MaxMHeapList]mSpanList // 页数在_MaxMHeapList(127)以内的使用的span链表
	busylarge mSpanList                // 页数 >= _MaxMHeapList使用的span链表
	sweepgen  uint32                   // sweep generation, see comment in mspan
	sweepdone uint32                   // all spans are swept
	sweepers  uint32                   // number of active sweepone calls

	// allspans is a slice of all mspans ever created. Each mspan
	// appears exactly once.
	//
	// The memory for allspans is manually managed and can be
	// reallocated and move as the heap grows.
	//
	// In general, allspans is protected by mheap_.lock, which
	// prevents concurrent access as well as freeing the backing
	// store. Accesses during STW might not hold the lock, but
	// must ensure that allocation cannot happen around the
	// access (since that may free the backing store).
	allspans []*mspan // all spans out there

	// sweepSpans contains two mspan stacks: one of swept in-use
	// spans, and one of unswept in-use spans. These two trade
	// roles on each GC cycle. Since the sweepgen increases by 2
	// on each cycle, this means the swept spans are in
	// sweepSpans[sweepgen/2%2] and the unswept spans are in
	// sweepSpans[1-sweepgen/2%2]. Sweeping pops spans from the
	// unswept stack and pushes spans that are still in-use on the
	// swept stack. Likewise, allocating an in-use span pushes it
	// on the swept stack.
	sweepSpans [2]gcSweepBuf

	//_ uint32 // align uint64 fields on 32-bit for atomics

	// Proportional sweep
	//
	// These parameters represent a linear function from heap_live
	// to page sweep count. The proportional sweep system works to
	// stay in the black by keeping the current page sweep count
	// above this line at the current heap_live.
	//
	// The line has slope sweepPagesPerByte and passes through a
	// basis point at (sweepHeapLiveBasis, pagesSweptBasis). At
	// any given time, the system is at (memstats.heap_live,
	// pagesSwept) in this space.
	//
	// It's important that the line pass through a point we
	// control rather than simply starting at a (0,0) origin
	// because that lets us adjust sweep pacing at any time while
	// accounting for current progress. If we could only adjust
	// the slope, it would create a discontinuity in debt if any
	// progress has already been made.
	pagesInUse         uint64  // pages of spans in stats _MSpanInUse; R/W with mheap.lock
	pagesSwept         uint64  // pages swept this cycle; updated atomically
	pagesSweptBasis    uint64  // pagesSwept to use as the origin of the sweep ratio; updated atomically
	sweepHeapLiveBasis uint64  // value of heap_live to use as the origin of sweep ratio; written with lock, read without
	sweepPagesPerByte  float64 // proportional sweep ratio; written with lock, read without
	// TODO(austin): pagesInUse should be a uintptr, but the 386
	// compiler can't 8-byte align fields.

	// Malloc stats.
	largealloc  uint64                  // bytes allocated for large objects
	nlargealloc uint64                  // number of large object allocations
	largefree   uint64                  // bytes freed for large objects (>maxsmallsize)
	nlargefree  uint64                  // number of frees for large objects (>maxsmallsize)
	nsmallfree  [_NumSizeClasses]uint64 // number of frees for small objects (<=maxsmallsize)

	// arenas is the heap arena map. It points to the metadata for
	// the heap for every arena frame of the entire usable virtual
	// address space.
	//
	// Use arenaIndex to compute indexes into this array.
	//
	// For regions of the address space that are not backed by the
	// Go heap, the arena map contains nil.
	//
	// Modifications are protected by mheap_.lock. Reads can be
	// performed without locking; however, a given entry can
	// transition from nil to non-nil at any time when the lock
	// isn't held. (Entries never transitions back to nil.)
	//
	// In general, this is a two-level mapping consisting of an L1
	// map and possibly many L2 maps. This saves space when there
	// are a huge number of arena frames. However, on many
	// platforms (even 64-bit), arenaL1Bits is 0, making this
	// effectively a single-level map. In this case, arenas[0]
	// will never be nil.
	arenas [1 << arenaL1Bits]*[1 << arenaL2Bits]*heapArena

	// heapArenaAlloc is pre-reserved space for allocating heapArena
	// objects. This is only used on 32-bit, where we pre-reserve
	// this space to avoid interleaving it with the heap itself.
	heapArenaAlloc linearAlloc

	// arenaHints is a list of addresses at which to attempt to
	// add more heap arenas. This is initially populated with a
	// set of general hint addresses, and grown with the bounds of
	// actual heap arena ranges.
	arenaHints *arenaHint

	// arena is a pre-reserved space for allocating heap arenas
	// (the actual arenas). This is only used on 32-bit.
	arena linearAlloc

	//_ uint32 // ensure 64-bit alignment of central

	// central free lists for small size classes.
	// the padding makes sure that the MCentrals are
	// spaced CacheLineSize bytes apart, so that each MCentral.lock
	// gets its own cache line.
	// central is indexed by spanClass.
	central [numSpanClasses]struct {
		mcentral mcentral
		pad      [sys.CacheLineSize - unsafe.Sizeof(mcentral{})%sys.CacheLineSize]byte
	} //每一个central对应一个spanclass

	spanalloc             fixalloc // 分配span的fixalloc
	cachealloc            fixalloc // 分配cache的fixalloc
	treapalloc            fixalloc // 为大对象分配树枝节点的fixalloc
	specialfinalizeralloc fixalloc // allocator for specialfinalizer*
	specialprofilealloc   fixalloc // allocator for specialprofile*
	speciallock           mutex    // lock for special record allocators.
	arenaHintAlloc        fixalloc // allocator for arenaHints

	unused *specialfinalizer // never set, just here to force the specialfinalizer type into DWARF
}
```

mheap_ 是一个全局变量，会在系统初始化的时候初始化（在函数 mallocinit() 中）

+ allspans []*mspan: 所有的 spans 都是通过 mheap_ 申请，所有申请过的 mspan 都会记录在 allspans。结构体中的 lock 就是用来保证并发安全的
+ central [numSpanClasses]struct {
		mcentral mcentral
		pad      [sys.CacheLineSize - unsafe.Sizeof(mcentral{})%sys.CacheLineSize]byte
	}:每种大小的块对应一个 pad 可以认为是一个字节填充，为了避免伪共享（false sharing）问题的
+ sweepgen, sweepdone: GC 相关
+ free      [_MaxMHeapList]mSpanList：这是一个 SpanList 数组，每个 SpanList 里面的 mspan 由 1 ~ 127 (_MaxMHeapList - 1) 个 page 组成。比如 free[3] 是由包含 3 个 page 的 mspan 组成的链表
+ spans []*mspan: 记录 arena 区域页号（page number）和 mspan 的映射关系
+ spanalloc, cachealloc fixalloc: fixalloc 是 free-list，用来分配特定大小的块
+ arena_start, arena_end, arena_used：
arena 是 Golang 中用于分配内存的连续虚拟地址区域。由 mheap 管理，堆上申请的所有内存都来自 arena，内存布局如下

```
+------------+----------------+------------------------------------+
|span 512mb  | bitmap  32gb   |  arena 512gb                       |   
+------------+----------------+------------------------------------+
|span_mapped | bitmap_mapped  |  arena_start  arena_used  arena_end|
+------------+----------------+------------------------------------+
```

#### 内存分配流程
1. 计算待分配对象所对应的规格
2. 从mcache.alloc数组中查找规格相同的object
3. 从mspan.freelist链表中提取可用的object
4. 若mspan.freelist为空，从mcentral更新获取mspan
5. 若mcentral.nonempty为空，从mheap.free/freelarge中获取并切分成object链表
6. 若heap没有大小合适的闲置span，向操作系统申请新的内存块

#### 内存释放流程
1. 将标记为可回收的object交还给mspan.freelist
2. 该mspan被释放会central，可供任意的mcache重新获取使用
3. 若mspan以回收全部object，则交还给mheap，以便重新切分
4. 定期扫描heap里长时间闲置的span，释放占用的内存

#### 分配流程（malloc.go文件中的说明）
1. 调整小对象，并查看p中mcache对应的mspan，扫描空的bitmap以找到空的slot，如果有空闲的则分配，这个过程可以在不分配锁的情况下进行
2. 如果没有空的slot，则从mcentral中获取所需大小，空闲的mspan，获取整个span以减少锁住mcentral的成本
3. 如果mcentral中的spanlist为空，则从mheap中获取mspan所需的一组页
4. 如果mheap为空或没有足够空间的页，则向操作系统获取一组页(至少1mb)，分配大量页面会减轻与操作系统通信的成本。

仅当mspan.needzero为false时，mspan中的空闲对象槽才会归零。如果needzero为true，则对象在分配时归零。以这种方式延迟归零有各种好处
1. 堆栈帧分配可以完全避免归零
2. 它表现出更好的时间局部性，因为程序可能要写入内存。
3. 不会让零页面永远不会被重用。