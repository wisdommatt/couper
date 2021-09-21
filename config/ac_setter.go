package config

type ErrorHandlerSetter struct {
	ErrorHandler []*ErrorHandler `hcl:"error_handler,block"`
}

func (ehs *ErrorHandlerSetter) Set(ehConf *ErrorHandler) {
	ehs.ErrorHandler = append(ehs.ErrorHandler, ehConf)
}
