package goof

import "errors"

var AlreadyMember = errors.New("already a member")
var NotMember = errors.New("not a member")
var NoSuchFilter = errors.New("no such filter")
var NoSuchAlliance = errors.New("no such alliance")
