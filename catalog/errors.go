// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

// Not Found
type NotFoundError struct{ s string }

func (e *NotFoundError) Error() string { return e.s }

// Conflict (non-unique id, assignment to read-only data)
type ConflictError struct{ s string }

func (e *ConflictError) Error() string { return e.s }

// Bad Request
type BadRequestError struct{ s string }

func (e *BadRequestError) Error() string { return e.s }
