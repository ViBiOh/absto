package model

import (
	"errors"
	"fmt"
	"io"
)

var errNotExists = errors.New("not exists")

func ErrNotExist(err error) error {
	return fmt.Errorf("%s: %w", err, errNotExists)
}

func IsNotExist(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, errNotExists)
}

func HandleClose(closer io.Closer, err error) error {
	if closeErr := closer.Close(); closeErr != nil {
		if err == nil {
			return closeErr
		}
		return fmt.Errorf("%s: %w", err, closeErr)
	}

	return err
}
