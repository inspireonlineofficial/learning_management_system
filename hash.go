package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	bytes, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), 10)
	fmt.Println(string(bytes))
}
