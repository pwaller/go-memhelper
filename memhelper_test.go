package memhelper

// TODO: Understand why go's garbage collector is refusing to let go of the last
//		 1GB Alloc()

import (
	"log"
	"math/rand"
	"testing"
	"time"
)

func Alloc(N ByteSize) []byte {
	result := make([]byte, int(N))
	log.Printf("Alloc(%v)", N)
	PrintStats()
	return result
}

func TestNeedRAM(t *testing.T) {
	log.Print("Testing.. ram = ", SpareMemory())
	var b []byte
	for i := 0; i < 10; i++ {
		nb := 1 * GB
		log.Printf("allocing %v", nb)
		//b = 
		Alloc(nb)
		log.Printf("deallocing %v", nb)
		b = []byte{}
	}
	log.Print("Testing.. ram = ", SpareMemory())

	Alloc(2*GB - 1*KB)
	time.Sleep(1 * time.Second)
	log.Printf("b length: %d", b)
}

func TestRAMBlocking(t *testing.T) {
	return
	done := make(chan bool)
	go func() {
		<-BlockUntilSpare(1*GB, 1*time.Second)
		///
		done <- true
	}()
	<-done
}

func TestOvercommit(t *testing.T) {
	defer func() {
		if x := recover(); x != nil {
			log.Print("x= ", x)
		}
	}()

	nrand := int(1e5)

	a := make([][]byte, 0)
	for i := 0; i < 15; i++ {
		b := Alloc(1 * GB)
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
	finish := make(chan bool)

	Nbyte := 100
	randomvalues := make([]byte, 0)
	randbyte := make(chan byte)
	go func() {
		for {
			randomvalues = append(randomvalues, <-randbyte)
		}
	}()

	bigalloc := func() {
		defer func() {
			if r := recover(); r != nil {
				log.Print("Recovered from panic: ", r)
			}
		}()
		a := Alloc(1 * GB)
		log.Printf("len(a) = %d", len(a))
		for i := 0; i < Nbyte; i++ {
			randbyte <- a[rand.Intn(int(len(a)))]
		}
		<-finish
	}

	N := 20
	for i := 0; i < N; i++ {
		go bigalloc()
		time.Sleep(1 * time.Second)
	}

	log.Print("Goroutines started..")
	time.Sleep(10 * time.Second)

	for i := 0; i < N; i++ {
		finish <- true
	}
}
