package wasmedge

// #include <wasmedge.h>
import "C"

type Validator struct {
	_inner *C.WasmEdge_ValidatorContext
}

func NewValidator() *Validator {
	self := &Validator{
		_inner: C.WasmEdge_ValidatorCreate(nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewValidatorWithConfig(conf *Configure) *Validator {
	self := &Validator{
		_inner: C.WasmEdge_ValidatorCreate(conf._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *Validator) Validate(ast *AST) error {
	return newError(C.WasmEdge_ValidatorValidate(self._inner, ast._inner))
}

func (self *Validator) Delete() {
	C.WasmEdge_ValidatorDelete(self._inner)
	self._inner = nil
}
