package main

import (
	"fmt"
	"os"
)

func validatePath(path string) error {

	if f, err := os.Stat(path); os.IsNotExist(err) || (err == nil && f.IsDir()) {
		if err != nil {
			return fmt.Errorf("not a valid socket file: %s", err.Error())
		}
		return fmt.Errorf("provided socket file is a directory")
	}

	return nil
}
