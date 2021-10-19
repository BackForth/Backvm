package main

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Invalid argument count.")
		os.Exit(1)
	}

	file = os.Args[1]
	data, error := ioutil.ReadFile(file)
	if error != nil {
		panic(error)
	}

}