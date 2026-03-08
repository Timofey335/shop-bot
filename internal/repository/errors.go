package repository

import "errors"

var ErrNotFound = errors.New("record not found")
var ErrDuplicate = errors.New("record already exists")
