package user

type ValidationError struct{ msg string }

func (e *ValidationError) Error() string { return e.msg }
