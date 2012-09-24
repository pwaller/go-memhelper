package memhelper

// TODO: Understand why go's garbage collector is refusing to let go of the last
//		 1GB Alloc()

import (
	"log"
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/pwaller/go-deathtest"
)

func Alloc(N ByteSize) []byte {
	result := make([]byte, int(N))
	log.Printf("Alloc(%.2b)", N)
	PrintStats()
	return result
}

func TestNeedRAM(t *testing.T) {
	return
	log.Print("Testing.. ram = ", SpareMemory())
	var b []byte
	for i := 0; i < 10; i++ {
		nb := 1 * GiB
		log.Printf("allocing %v", nb)
		Alloc(nb)
		log.Printf("deallocing %v", nb)
		b = []byte{}
		PrintStats()
		runtime.GC()
		PrintStats()
		_ = b
	}
}

func TestRAMBlocking(t *testing.T) {
	return
}

func TestOvercommit(t *testing.T) {
	return
	if !deathtest.Run(t) {
		return
	}
	defer func() {
		if x := recover(); x != nil {
			log.Print("x= ", x)
		}
	}()

	nrand := int(1e5)

	a := make([][]byte, 0)
	for i := 0; i < 15; i++ {
		b := Alloc(1 * GiB)
		for i := 0; i < nrand; i++ {
			b[rand.Intn(int(len(b)))] = byte(rand.Uint32() % 8)
		}
		PrintProcStat()
		a = append(a, b)
	}
	log.Print("--here")
	PrintStats()
	time.Sleep(60 * time.Second)
}

func TestFailure(t *testing.T) {
	if !deathtest.Run(t) {
		return
	}
	finish := make(chan bool)

	bigalloc := func() {
		//runtime.GC()
		a := Alloc(1 * GiB)
		for i := 0; i < len(a); i += int(4 * kiB) {
			//a[rand.Intn(int(len(a)))] = 1
			a[i] = 1
		}
		<-finish
		//log.Print("a[100] = ", a[100])
	}

	N := 20
	for i := 0; i < N; i++ {
		go bigalloc()
		time.Sleep(1000 * time.Millisecond)
	}

	log.Print("Goroutines started..")
	time.Sleep(10 * time.Second)

	for i := 0; i < N; i++ {
		finish <- true
	}

	log.Print("MaxRSS: ", GetMaxRSS())
}
