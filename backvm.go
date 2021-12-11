package main

// #cgo CFLAGS: -g -Wall
// #include <stdlib.h>
// #include "back.h"
import "C"
import (
	"os"
	"io/ioutil"
	"fmt"
	"strings"
	"strconv"
	"sync"
	"unsafe"
)

type state struct {
	bytecode []int
	id int
	vars map[int]int
	stack *[]int
	iptr int
	lptr int
	lpidx int
	max int
	skip bool
	inloop bool
}

type BackMutex struct {
	unlocked bool
	mutex sync.Mutex
}

func pop(list *[]int) int {
	var x int
	tm := *list
	if len(tm) == 0 { //couldn't !len(tm) because go is weird
		panic("Not enough Elements in stack to perform operation")
	}
	x, *list = tm[len(tm)-1], tm[:len(tm)-1]
	return x
}

func push(list *[]int, item int) {
	*list = append(*list, item)
}

func recv(stack *[]int, chid int, mtx *BackMutex) {
	*stack = append(*stack, <-channelArr[chid])
	(*mtx).mutex.Unlock()
	(*mtx).unlocked = true
}

func getptr(stack *[]int) (bool,uintptr) {
	address := fmt.Sprintf("0x%x", pop(stack))
	adr, err := strconv.ParseInt(address, 0, 64)
	if err != nil {
		*stack = append(*stack, 1)
		return false, 0
	}
	return true, uintptr(adr)
}

func unsp(a uintptr) unsafe.Pointer {
	return unsafe.Pointer(a)
}

func intp(a uintptr) *int {
	return (*int)(unsafe.Pointer(a))
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

var mutexArr []BackMutex
var channelArr []chan int
var waiter sync.WaitGroup

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

func initState(bytecode []int, id int, stat *state) {
	tm := state{}
	tm.bytecode = bytecode
	tm.id = id
	tm.vars = make(map[int]int)
	tm.stack = &([]int{})
	tm.lptr = 0
	tm.lpidx = 0
	tm.max = 0
	tm.skip = false
	tm.inloop = false
	(*stat) = tm
}

var fncm map[int]func(*state) = map[int]func(*state){
	0: func(*state){},
	1: func(s *state) {
		(*s).iptr++
		push((*s).stack, (*s).bytecode[(*s).iptr])
	},
	2: func(s *state) {
		fmt.Print(pop((*s).stack))
	},
	3: func(s *state) {
		var in int
		fmt.Scanf("%d", &in)
		push((*s).stack, in)
	},
	4: func(s *state) {
		fmt.Print(string(pop((*s).stack)))
	},
	5: func(s *state) {
		push((*s).stack, pop((*s).stack)+pop((*s).stack))
	},
	6: func(s *state) {
		a := pop((*s).stack)
		push((*s).stack, pop((*s).stack)-a)
	},
	7: func(s *state) {
		push((*s).stack, pop((*s).stack)*pop((*s).stack))
	},
	8: func(s *state) {
		a := pop((*s).stack)
		push((*s).stack, pop((*s).stack)/a)
	},
	9: func(s *state) {
		a := pop((*s).stack)
		push((*s).stack, pop((*s).stack)%a)
	},
	10: func(s *state) {
		if pop((*s).stack) == 0 {
			(*s).skip = true
		}
	},
	11: func(s *state) {
		(*s).skip = false
	},
	12: func(s *state) {
		a := pop((*s).stack)
		push((*s).stack, a)
		push((*s).stack, a)
	},
	13: func(s *state) {
		x3 := pop((*s).stack)
		x2 := pop((*s).stack)
		x1 := pop((*s).stack)
		push((*s).stack, x2)
		push((*s).stack, x3)
		push((*s).stack, x1)
	},
	14: func(s *state) {
		a, b := pop((*s).stack), pop((*s).stack)
		push((*s).stack, a)
		push((*s).stack, b)
	},
	15: func(s *state) {
		pop((*s).stack)
	},
	16: func(s *state) {
		a, b := pop((*s).stack), pop((*s).stack)
		push((*s).stack, b)
		push((*s).stack, a)
		push((*s).stack, b)
	},
	17: func(s *state) {
		val := int(C.alloc(C.int(pop((*s).stack))))
		push((*s).stack, val)
	},
	18: func(s *state) {
		reslt, ad := getptr((*s).stack)
		p := unsp(ad)
		if reslt {
			C.free(p)
		}
	},
	19: func(s *state) {
		reslt, ad := getptr((*s).stack)
		p := intp(ad)
		if reslt {	
			*p = pop((*s).stack)
		}
	},
	20: func(s *state) {
		reslt, ad := getptr((*s).stack)
		p := intp(ad)
		if reslt {
			push((*s).stack, *p)
		}
	},
	21: func(s *state) {
		val, idx := pop((*s).stack), pop((*s).stack)
		waiter.Add(1)
		go func(mutex *BackMutex, msg chan int){
			(*mutex).mutex.Lock()
			(*mutex).unlocked = false
			msg <- val
			waiter.Done()
		}(&mutexArr[idx], channelArr[idx])
	},
	22: func(s *state) {
		mutex := &mutexArr[(*s).id]
		for {
			if !(*mutex).unlocked {
				break
			}
		}
		recv((*s).stack, (*s).id, mutex)
	},
	23: func(s *state) {
		mutex, num := &mutexArr[(*s).id], pop((*s).stack)
		for counter := 0; counter < (num);counter++ {
			if (*mutex).unlocked {
				continue
			}
			recv((*s).stack, (*s).id, mutex)
		}
	},
	24: func(s *state) {
		os.Exit(pop((*s).stack))
	},
	25: func(s *state) {
		start, end := pop((*s).stack), pop((*s).stack)
		(*s).inloop = true
		(*s).max = end
		(*s).lptr = start
		(*s).lpidx = (*s).iptr
	},
	26: func(s *state) {
		if !(*s).inloop {
			panic("Encounted `loop` word without previous `do` word.")
		}
		if (*s).lptr == (*s).max-1 {
			(*s).inloop = false
			return//continue
		}
		(*s).iptr = (*s).lpidx
		(*s).lptr++
	},
	27: func(s *state) {
		(*s).iptr++
		(*s).vars[(*s).bytecode[(*s).iptr]] = pop((*s).stack)
	},
	28: func(s *state) {
		(*s).iptr++
		push((*s).stack, (*s).vars[(*s).bytecode[(*s).iptr]])
	},
}

func execute(bytecode []int, id int) {
	machine := state{}
	initState(bytecode, id, &machine)
	for machine.iptr = 0;machine.iptr < len(bytecode);machine.iptr++ {
		if machine.skip {
			if bytecode[machine.iptr] == 11 {
				fncm[9](&machine)
			}
			continue
		}
		fncm[bytecode[machine.iptr]](&machine)
	}
	waiter.Done()
}
