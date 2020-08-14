package handler

import (
	"net/http"

	ac "go.avenga.cloud/couper/gateway/access_control"
	"go.avenga.cloud/couper/gateway/errors"
)

type AccessControl struct {
	ac        ac.List
	errorTpl  *errors.Template
	protected http.Handler
}

func NewAccessControl(protected http.Handler, errTpl *errors.Template, list ...ac.AccessControl) *AccessControl {
	return &AccessControl{
		ac:        list,
		errorTpl:  errTpl,
		protected: protected,
	}
}

func (a *AccessControl) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	for _, control := range a.ac {
		if err := control.Validate(req); err != nil {
			var code errors.Code
			switch err {
			case ac.ErrorNotConfigured:
				code = errors.ConfigurationError
			case ac.ErrorEmptyToken:
				code = errors.AuthorizationRequired
			default:
				code = errors.AuthorizationFailed
			}
			a.errorTpl.ServeError(code).ServeHTTP(rw, req)
			return
		}
	}
	a.protected.ServeHTTP(rw, req)
}

func (a *AccessControl) String() string {
	if h, ok := a.protected.(interface{ String() string }); ok {
		return h.String()
	}
	return "AccessControl"
}
