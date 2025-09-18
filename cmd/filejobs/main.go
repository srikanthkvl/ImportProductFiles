package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main(){
	f, err := os.OpenFile(filepath.Join("./sample/users.csv"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer f.Close()
	
	for i := 1; i < 50000; i++ {
		if _, err := f.WriteString(fmt.Sprintf("%d,foo%d,foo,foo,foo%d@foo.com,%d\n", i, i, i, i)); err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
	}
	fmt.Println("Successfully appended to file")
}