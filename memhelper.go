package memhelper

// Package to try and avoid running out of memory

// TODO: Docs
//       Test
//       Be a bit more efficient?
//       First functional implementation of memhelper.BlockUnlessSpare

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var GRACE_ABS = flag.Int64("memhelper.abs", 100, "absolute ram to leave left over (MB)")
var GRACE_REL = flag.Int64("memhelper.rel", 20, "(%%) of free memory at program start to leave")
var debug = flag.Bool("memhelper.debug", false, "print debug messages for needram")

var GC_EVERY = flag.Duration("memhelper.gc", 0, "force garbage collection (0 = off)")

// Amount of memory which was spare when the program started
var SPARE_AT_PROGRAM_START ByteSize = SystemSpareMemory()

func init() {
	flag.Parse()
	if *GC_EVERY != 0 {
		if *debug {
			log.Printf("Will GC() every %v", *GC_EVERY)
		}
		go func() {
			for {
				time.Sleep(*GC_EVERY)
				if *debug {
					log.Print("GC()")
				}
				runtime.GC()
				if *debug {
					GCStats()
					SpareMemory()
				}
			}
		}()
	}

	go ProcessRequests()
}

// A pretty-printing/parsing number of bytes
// TODO: Implement parsing (so we satisfy flag.Var)
// This is done to avoid repeatedly allocating these structures
var memstats *runtime.MemStats = new(runtime.MemStats)
var sysinfo *syscall.Sysinfo_t = new(syscall.Sysinfo_t)

// Not sure if the mutex is the most efficient thing to be doing here, but
// let's run with it for the moment
var memstats_mutex, sysinfo_mutex *sync.RWMutex = new(sync.RWMutex), new(sync.RWMutex)

func update_sysinfo() {
	sysinfo_mutex.Lock()
	defer sysinfo_mutex.Unlock()
	err := syscall.Sysinfo(sysinfo)
	if err != nil {
		log.Panic("syscall.Sysinfo failed: ", err)
	}
}

// Return the total amount of memory (including caches) on the system
func SystemSpareMemory() ByteSize {
	update_sysinfo()
	sysinfo_mutex.RLock()
	defer sysinfo_mutex.RUnlock()
	return ByteSize(sysinfo.Freeram + sysinfo.Bufferram)
}

// 
func update_memstats() {
	memstats_mutex.Lock()
	defer memstats_mutex.Unlock()
	runtime.ReadMemStats(memstats)
}

// Prints information about the most recent GC
func GCStats() {
	update_memstats()
	memstats_mutex.RLock()
	defer memstats_mutex.RUnlock()
	log.Printf("     -- paused for %v -- total %v -- N %d",
		time.Duration(memstats.PauseNs[(memstats.NumGC-1)%256]),
		time.Duration(memstats.PauseTotalNs), memstats.NumGC)
}

// Memory Go owns but isn't using
func GoSpareMemory() ByteSize {
	update_memstats()
	memstats_mutex.RLock()
	defer memstats_mutex.RUnlock()
	return ByteSize(memstats.HeapIdle)
}

// Total amount of memory go owns from the system (Virtual memory)
func GoTotalUsed() ByteSize {
	update_memstats()
	memstats_mutex.RLock()
	defer memstats_mutex.RUnlock()
	return ByteSize(memstats.Sys)
}

// Print stats obtained 
func PrintStats() {
	update_memstats()
	memstats_mutex.RLock()
	defer memstats_mutex.RUnlock()
	log.Printf("   Alloc    Total      Sys  | Heap-> |   Alloc      Sys     Idle    Inuse    Relsd        N")
	log.Printf("%8v %8v %8v            %8v %8v %8v %8v %8v %8d",
		ByteSize(memstats.Alloc), ByteSize(memstats.TotalAlloc),
		ByteSize(memstats.Sys), ByteSize(memstats.HeapAlloc),
		ByteSize(memstats.HeapSys), ByteSize(memstats.HeapIdle),
		ByteSize(memstats.HeapInuse),
		ByteSize(memstats.HeapReleased), memstats.HeapObjects)

	fmt.Println()
}

// Returns the number of spare megabytes of ram after leaving 100 + 10% spare
func SpareMemory() ByteSize {
	// An amount of memory to leave spare
	grace := ByteSize((*GRACE_REL)*int64(SPARE_AT_PROGRAM_START)/100 + (*GRACE_ABS)*int64(MB))

	free := SystemSpareMemory()
	allocated_but_unused := GoSpareMemory()

	spare := ByteSize((free - grace) + allocated_but_unused)

	if *debug {
		log.Printf("Spare : %7v = %7v - %7v | Go Spare/Total %7v / %7v",
			spare, free, grace, allocated_but_unused, GoTotalUsed())
		PrintStats()
	}

	return spare
}

// A memory request
type Request struct {
	start     time.Time     // Start time of signal (so that we can compute the .Since)
	duration  time.Duration // Length of time we think we need it for
	amount    ByteSize      // Amount of memory to reserve
	satisfied chan bool     // Used to signal the requester that they can proceed
}

// Signal the request handler that someone wants memory
var request chan Request = make(chan Request)

// A constant used to internally by the processor to schedule the thinking needed
var request_sentinel Request = Request{amount: -1}

// A permanently running goroutine to service requests to reserve memory
// TODO: Finish the implementation
//       Need to maintain a list of outstanding memory requests, and probably
//         try and permit smaller allocations immediately
func ProcessRequests() {
	var outstanding, unsatisfied []Request

	_ = unsatisfied

	// Generate sentinels to cause processing when new requests aren't coming
	go func() {
		for {
			request <- request_sentinel
			// TODO: Adjust timer
			time.Sleep(1000 * time.Millisecond)
		}
	}()

	process_outstanding := func() {
		if *debug {
			//log.Printf("%d outstanding requests", len(outstanding))
		}
		for _, m := range outstanding {
			_ = m
		}
	}

	for r := range request {
		if r == request_sentinel {
			process_outstanding()
		}
	}
}

// Concept: attempt to reserve an `amount` amount of memory for `duration`. 
// If it's unavailable, wait until it is. During `duration`, `amount` is
// subtracted from the available memory to prevent multiple goroutines
// overcommitting.
// Example:
//   used := <-BlockUntilSpare(10*memhelper.MB, 10*time.Millisecond)
//   make([]byte, 10*1024*1024)
//   used <- true
func BlockUntilSpare(amount ByteSize, duration time.Duration) <-chan bool {
	r := Request{amount: amount, duration: duration, satisfied: make(chan bool)}
	request <- r
	return r.satisfied
}

func PrintProcStat() {
	log.Print("GOOS: ", runtime.GOOS)
	if runtime.GOOS != "linux" {
		log.Panic("PrintProcStat() not implemented for systems other than linux")
	}
	fd, err := os.Open("/proc/self/stat")
	if err != nil {
		panic(err)
	}
	bytes, err := ioutil.ReadAll(fd)
	if err != nil {
		panic(err)
	}
	s := string(bytes)
	parts := strings.Split(s, " ")
	rss_pages, err := strconv.ParseInt(parts[23], 10, 64)
	if err != nil {
		panic(err)
	}
	vss, err := strconv.ParseInt(parts[22], 10, 64)
	if err != nil {
		panic(err)
	}
	log.Printf("Stat: %v %v", ByteSize(vss), ByteSize(rss_pages*4096))
}
