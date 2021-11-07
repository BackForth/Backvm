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

var vars map[int]int
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
	address := fmt.Sprintf("0x%x", pop(stack))
	adr, err := strconv.ParseInt(address, 0, 64)
	if err != nil {
		panic(err)
		*stack = append(*stack, 1)
		res.success = false
		return res
	}
	res.ptr = (*int)(unsafe.Pointer(uintptr(adr)))
	return res
}

func getuptr(stack *[]int) UResult {
	res := UResult{}
	res.success = true
	address := fmt.Sprintf("0x%x", pop(stack))
	var adr uint64
	adr, err := strconv.ParseUint(address, 0, 64)
	if err != nil {
		*stack = append(*stack, 1)
		adr = 0
		res.success = false
	}
	res.ptr = unsafe.Pointer(uintptr(adr))
	return res
}

func execute(bytecode []int, id int) {
	var stack []int
	var iptr, lptr, lpidx, max int
	var skip, inloop bool
	for iptr = 0; iptr < len(bytecode); iptr++ {
		if skip {
			if bytecode[iptr] == 9 {
				skip = false
			}
			continue
		}
		switch bytecode[iptr] {
			case 1:
				fmt.Print(pop(&stack))
			case 2:
				var in int
				fmt.Scanf("%d", &in)
				stack = append(stack, in)
			case 3:
				fmt.Print(string(pop(&stack)))
			case 4:
				stack = append(stack, pop(&stack)+pop(&stack))
			case 5:
				a := pop(&stack)
				stack = append(stack, pop(&stack)-a)
			case 6:
				stack = append(stack, pop(&stack)*pop(&stack))
			case 7:
				a := pop(&stack)
				stack = append(stack, pop(&stack)/a)
			case 8:
				a := pop(&stack)
				stack = append(stack, pop(&stack)%a)
			case 9:
				val := pop(&stack)
				if val == 0 {
					skip = true
				}
			case 10:
				skip = false
			case 11:
				a := pop(&stack)
				stack = append(append(stack, a), a)
			case 12:
				x3 := pop(&stack)
				x2 := pop(&stack)
				x1 := pop(&stack)
				stack = append(append(append(stack, x2), x3), x1)
			case 13:
				a, b := pop(&stack), pop(&stack)
				stack = append(append(stack, a), b)
			case 14:
				pop(&stack)
			case 15:
				a, b := pop(&stack), pop(&stack)
				stack = append(append(append(stack, b), a), b)
			case 16:
				val := int(C.alloc(C.int(pop(&stack))))
				stack = append(stack, val)

			case 17:
				reslt := getuptr(&stack)
				if reslt.success {
					C.free(reslt.ptr)
				}
			case 18:
				reslt := getptr(&stack) 
				if reslt.success {
					*reslt.ptr = pop(&stack)
				}	
			case 19:
				reslt := getptr(&stack)
				if reslt.success {
					stack = append(stack, *reslt.ptr)
				}
			case 20:
				val, idx := pop(&stack), pop(&stack)
				waiter.Add(1)
				go func(mutex *BackMutex, msg chan int){
					(*mutex).mutex.Lock()
					(*mutex).unlocked = false
					(*mutex).sender = id
					msg <- val
					waiter.Done()
				}(&mutexArr[idx], channelArr[idx])
			case 21:
				mutex := &mutexArr[id]
				for {
					if !(*mutex).unlocked {
						break
					}
				}
				recv(&stack, id, mutex)
			case 22:
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
			case 23:
				os.Exit(pop(&stack))
			case 24:
				start, end := pop(&stack), pop(&stack)
				inloop = true
				max = end
				lptr = start
				lpidx = iptr
			case 25:
				if !inloop {
					panic("Encounted `loop` word without previous `do` word.")
				}
				if lptr == max-1 {
					inloop = false
					continue
				}
				iptr = lpidx
				lptr++
			case 26:
				iptr++
				stack = append(stack, bytecode[iptr])
			case 27:
				iptr++
				vars[bytecode[iptr]] = pop(&stack)
			case 28:
				iptr++
				stack = append(stack, vars[bytecode[iptr]])
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
	vars = make(map[int]int)

	var tptr int
	fmt.Println()
	for tptr = 0; tptr < len(threads); tptr++ {
		waiter.Add(1)
		go execute(threads[tptr], tptr)
	}
	waiter.Wait()
}
