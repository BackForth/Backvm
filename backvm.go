package main

// #cgo CFLAGS: -g -Wall
// #include <stdlib.h>
// #include "back.h"
import "C"
import (
	"os"
	"fmt"
	"io/ioutil"
	"strings"
	"strconv"
	"sync"
	"unsafe"
)

type BackMutex struct {
	unlocked bool
	mutex sync.Mutex
	sender int
}

type Result struct {
	success bool
	ptr *int
}

type UResult struct {
	success bool
	ptr unsafe.Pointer
}

var mutexArr []BackMutex
var channelArr []chan int
var waiter sync.WaitGroup

func pop(list *[]int) int {
	var x int
	tm := *list
	if len(tm) == 0 { //couldn't !len(tm) because go is weird
		panic("Not enough Elements in stack to perform operation")
	}
	x, *list = tm[len(tm)-1], tm[:len(tm)-1]
	return x
}

func recv(stack *[]int, chid int, mtx *BackMutex) {
	*stack = append(*stack, <-channelArr[chid])
	(*mtx).mutex.Unlock()
	(*mtx).unlocked = true
	(*mtx).sender = -1
}

func getptr(stack *[]int) Result {
	res := Result{}
	res.success = true
	address := fmt.Sprintf("%x", pop(stack))
	var adr uint64
	adr, err := strconv.ParseUint(address, 0, 64)
	if err != nil {
		*stack = append(*stack, 1)
		adr = 0
		res.success = false
	}
	var ptr uintptr = uintptr(adr)
	res.ptr = (*int)(unsafe.Pointer(ptr))
	return res
}

func getuptr(stack *[]int) UResult {
	res := UResult{}
	res.success = true
	address := fmt.Sprintf("%x", pop(stack))
	var adr uint64
	adr, err := strconv.ParseUint(address, 0, 64)
	if err != nil {
		*stack = append(*stack, 1)
		adr = 0
		res.success = false
	}
	var ptr uintptr = uintptr(adr)
	res.ptr = unsafe.Pointer(ptr)
	return res
}

func execute(bytecode []int, id int) {
	var stack []int
	var iptr int
	var skip bool
	for iptr = 0; iptr < len(bytecode); iptr++ {
		if skip {
			if bytecode[iptr] == 9 {
				skip = false
			}
			continue
		}
		switch bytecode[iptr] {
			case 0:
				fmt.Print(pop(&stack))
			case 1:
				var in int
				fmt.Scanf("%d", &in)
				stack = append(stack, in)
			case 2:
				fmt.Print(string(pop(&stack)))
			case 3:
				stack = append(stack, pop(&stack)+pop(&stack))
			case 4:
				a := pop(&stack)
				stack = append(stack, pop(&stack)-a)
			case 5:
				stack = append(stack, pop(&stack)*pop(&stack))
			case 6:
				a := pop(&stack)
				stack = append(stack, pop(&stack)/a)
			case 7:
				a := pop(&stack)
				stack = append(stack, pop(&stack)%a)
			case 8:
				val := pop(&stack)
				if val == 0 {
					skip = true
				}
			case 10:
				a := pop(&stack)
				stack = append(append(stack, a), a)
			case 11:
				x3 := pop(&stack)
				x2 := pop(&stack)
				x1 := pop(&stack)
				stack = append(append(append(stack, x2), x3), x1)
			case 12:
				a, b := pop(&stack), pop(&stack)
				stack = append(append(stack, a), b)
			case 13:
				pop(&stack)
			case 14:
				a, b := pop(&stack), pop(&stack)
				stack = append(append(append(stack, b), a), b)
			case 15:
				val := int(C.alloc(C.int(pop(&stack))))
				stack = append(stack, val)
			case 16:
				reslt := getuptr(&stack)
				if reslt.success {
					C.free(reslt.ptr)
				}
			case 17:
				val, reslt := pop(&stack), getptr(&stack)
				if reslt.success {
					*reslt.ptr = val
				}
			case 18:
				reslt := getptr(&stack)
				if reslt.success {
					stack = append(stack, *reslt.ptr)
				}
			case 19:
				val, idx := pop(&stack), pop(&stack)
				waiter.Add(1)
				go func(mutex *BackMutex, msg chan int){
					(*mutex).mutex.Lock()
					(*mutex).unlocked = false
					(*mutex).sender = id
					msg <- val
					waiter.Done()
				}(&mutexArr[idx], channelArr[idx])
			case 20:
				mutex := &mutexArr[id]
				for {
					if !(*mutex).unlocked {
						break
					}
				}
				recv(&stack, id, mutex)
			case 21:
				os.Exit(pop(&stack))
			case 22:
				iptr++
				stack = append(stack, bytecode[iptr])
			case 23:
				mutex, num := &mutexArr[id], pop(&stack)
				counter := 0
				for {
					if counter == num {
						break
					}
					mtx := *mutex
					if mtx.unlocked {
						continue
					}
					recv(&stack, id, mutex)
					counter += 1
				}
		}
	}
	waiter.Done()
}

func parse(src []string) [][]int {
	var sptr int
	var code_list []int
	var codes [][]int
	for sptr = 0; sptr < len(src); sptr++ {
		op, err := strconv.Atoi(src[sptr])
		if err != nil {
			tmp := BackMutex{}
			tmp.unlocked = true
			mutexArr = append(mutexArr, tmp)
			channelArr = append(channelArr, make(chan int))
			if len(code_list) != 0 {
				codes = append(codes, code_list)
				code_list = []int{}
			}
		} else {
			code_list = append(code_list, op)
		}
	}
	codes = append(codes, code_list)
	return codes
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Invalid argument count.")
		os.Exit(1)
	}

	file := os.Args[1]
	data, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	threads := parse(strings.Fields(string(data)))

	var tptr int
	fmt.Println()
	for tptr = 0; tptr < len(threads); tptr++ {
		waiter.Add(1)
		go execute(threads[tptr], tptr)
	}
	waiter.Wait()
}
