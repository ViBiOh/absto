package model

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	errNotExists      = errors.New("not exists")
	ErrRelativePath   = errors.New("pathname contains relatives paths")
	ErrInvalidPath    = errors.New("pathname is invalid")
	relativePathRegex = regexp.MustCompile(`(?m)(\/|^)\.\.(\/|$)`)
)

func ErrNotExist(err error) error {
	return fmt.Errorf("%s: %w", err, errNotExists)
}

func IsNotExist(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, errNotExists)
}

func ValidPath(pathname string) error {
	if relativePathRegex.MatchString(pathname) {
		return ErrRelativePath
	}

	return nil
}
