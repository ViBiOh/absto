package model

import (
	"errors"
	"regexp"
)

var (
	errNotExists      = errors.New("not exists")
	ErrRelativePath   = errors.New("name contains relatives paths")
	ErrInvalidPath    = errors.New("name is invalid")
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
