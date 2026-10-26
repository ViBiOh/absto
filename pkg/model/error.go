package model

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	errNotExists      = errors.New("not exists")
	ErrNotEnabled     = errors.New("not enabled")
	ErrRelativePath   = errors.New("name contains relatives paths")
	ErrInvalidPath    = errors.New("name is invalid")
	ErrDirExpected    = errors.New("directory expected")
	ErrUnsupported    = fmt.Errorf("not supported by the storage: %w", errors.ErrUnsupported)
	relativePathRegex = regexp.MustCompile(`(?m)(\/|^)\.\.(\/|$)`)
)

func ErrNotExist(err error) error {
	return errors.Join(err, errNotExists)
}

func IsNotExist(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, errNotExists)
}

func ValidPath(name string) error {
	if relativePathRegex.MatchString(name) {
		return ErrRelativePath
	}

	return nil
}
