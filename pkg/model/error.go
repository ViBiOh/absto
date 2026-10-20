package model

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
)

type notExistError struct{}

func (notExistError) Error() string {
	return "not exists"
}

func (notExistError) Is(target error) bool {
	return target == fs.ErrNotExist
}

var (
	errNotExists      error = notExistError{}
	ErrRelativePath         = errors.New("name contains relatives paths")
	ErrInvalidPath          = errors.New("name is invalid")
	ErrUnsupported          = fmt.Errorf("not supported by the storage: %w", errors.ErrUnsupported)
	relativePathRegex       = regexp.MustCompile(`(?m)(\/|^)\.\.(\/|$)`)
)

func ErrNotExist(err error) error {
	if err == nil {
		return errNotExists
	}

	if IsNotExist(err) {
		return err
	}

	return fmt.Errorf("%w: %w", err, errNotExists)
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
