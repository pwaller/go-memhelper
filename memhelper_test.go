package memhelper

// TODO: Understand why go's garbage collector is refusing to let go of the last
//		 1GB Alloc()

import (
	"log"
	"testing"
	"time"
)

func Alloc(N ByteSize) []byte {
	return make([]byte, 0, int(N))
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
	time.Sleep(10 * time.Second)
	log.Printf("b length: %d", b)
}

func TestRAMBlocking() {
	done := make(chan bool)
	go func() {
		<-BlockUntilSpare(1*GB, 1*time.Second)
		///
		done <- 1
	}()
	<-done
}
