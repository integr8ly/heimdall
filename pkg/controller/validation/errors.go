package validation

type ParseErr struct {
	Message string
}

func (pe *ParseErr)Error()string  {
	return pe.Message
}

func IsParseErr(err error)bool{
	_,ok := err.(*ParseErr)
	return ok
}
