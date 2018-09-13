package p2p

import (
	"fmt"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestMessageQueue1(t *testing.T) {
	mq := NewMessageQueue(50)

	//test adding 20 elements
	i := 0
	for i < 20 {
		mq.Add(strconv.Itoa(i))

		i++
	}

	if mq.Len() != 20 {
		t.Fatal("Should have been 20 elements")
	}

	//test exists all 20 elements
	i = 0
	for i < 20 {
		if !mq.Contains(strconv.Itoa(i)) {
			t.Fatal("Should have found the" + strconv.Itoa(i) + "element!")
		}

		i++
	}

	//test if the 21-st element does not exists
	if mq.Contains(strconv.Itoa(i)) {
		t.Fatal("Should have not found the" + strconv.Itoa(i) + "element!")
	}

	//test adding 51 elements. It should be 50 elements
	i = 0
	for i < 51 {
		mq.Add(strconv.Itoa(i))

		i++
	}

	if mq.Len() != 50 {
		t.Fatal("Should have been 50 elements")
	}

	i = 1
	for i < 51 {
		if !mq.Contains(strconv.Itoa(i)) {
			t.Fatal("Should have found the" + strconv.Itoa(i) + "element!")
		}

		i++
	}

	//test if the 0-th element does not exists
	if mq.Contains(strconv.Itoa(0)) {
		t.Fatal("Should have not found the" + strconv.Itoa(0) + "element!")
	}

}

func TestMessageQueueMemLeak(t *testing.T) {
	timeStart := time.Now()
	timeIntermed := time.Now()

	PrintMemUsage()

	i := 0

	mq := NewMessageQueue(5000)

	for {
		mq.Add(strconv.Itoa(i))

		if time.Now().Sub(timeStart) > time.Second*10 {
			return
		}

		if time.Now().Sub(timeIntermed) >= time.Second {
			timeIntermed = time.Now()
			PrintMemUsage()
		}

		i++
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	if bToMb(m.Alloc) > 5 {
		t.Fatal("Allocated memory should have been less than 5 MiB!")
	}

}

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %4d MiB | ", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %4d MiB | ", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %4d MiB | ", bToMb(m.Sys))
	fmt.Printf("\tFreed objs = %9d | ", m.Frees)
	fmt.Printf("\tNumGC = %2d\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func TestCleanEmptyQueue(t *testing.T) {
	mq1 := NewMessageQueue(0)

	mq1.clean()

	if mq1.Len() != 0 {
		t.Error("mq1 should have had a length of 0!")
	}
}