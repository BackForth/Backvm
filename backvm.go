package main

import (
	"os"
	"fmt"
	"io/ioutil"
	"strings"
	"strconv"
	"sync"
)

var channel_mapper map[int]chan int
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

func backsend(val int, id int) {
	channel_mapper[id] <- val
	waiter.Done()
}

func execute(bytecode []int) {
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
			/*case 15:
				//
			case 16:
				//
			case 17:
				//
			case 18:
				//*/
			case 19:
				waiter.Add(1)
				go backsend(pop(&stack), pop(&stack))
			case 20:
				stack = append(stack, <-channel_mapper[pop(&stack)])
			case 21:
				os.Exit(pop(&stack))
			case 22:
				iptr++
				stack = append(stack, bytecode[iptr])
		}
	}
	waiter.Done()
}

func parse(src []string) [][]int {
	channel_mapper := make(map[int]chan int)

	var sptr int
	var code_list []int
	var codes [][]int
	cptr := 0
	for sptr = 0; sptr < len(src); sptr++ {
		op, err := strconv.Atoi(src[sptr])
		if err != nil {
			channel_mapper[cptr] = make(chan int)
			cptr++
			if len(code_list) != 0 {
				codes = append(codes, code_list)
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
	for tptr = 0; tptr < len(threads); tptr++ {
		waiter.Add(1)
		go execute(threads[tptr])
	}
	waiter.Wait()
}
