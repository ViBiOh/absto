package model

import (
	"errors"
	"fmt"
)

var errNotExists = errors.New("not exists")

// ErrNotExist create a NotExist error
func ErrNotExist(err error) error {
	return fmt.Errorf("%s: %w", err, errNotExists)
}

// IsNotExist checks if error match a not found
func IsNotExist(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, errNotExists)
}

// HandleClose call properly
func HandleClose(closer Closer, err error) error {
	if closeErr := closer(); closeErr != nil {
		if err == nil {
			return closeErr
		}
		return fmt.Errorf("%s: %w", err, closeErr)
	}

	return err
}
