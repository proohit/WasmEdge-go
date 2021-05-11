package wasmedge

/*
#include <wasmedge.h>
size_t _GoStringLen(_GoString_ s);
const char *_GoStringPtr(_GoString_ s);
*/
import "C"
import (
	"io/ioutil"
	"os"
	"unsafe"
)

type VM struct {
	_inner *C.WasmEdge_VMContext
}

func NewVM() *VM {
	self := &VM{
		_inner: C.WasmEdge_VMCreate(nil, nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewVMWithConfig(conf *Configure) *VM {
	self := &VM{
		_inner: C.WasmEdge_VMCreate(conf._inner, nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewVMWithStore(store *Store) *VM {
	self := &VM{
		_inner: C.WasmEdge_VMCreate(nil, store._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewVMWithConfigAndStore(conf *Configure, store *Store) *VM {
	self := &VM{
		_inner: C.WasmEdge_VMCreate(conf._inner, store._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *VM) RegisterWasmFile(modname string, path string) error {
	modstr := toWasmEdgeStringWrap(modname)
	var cpath = C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	return newError(C.WasmEdge_VMRegisterModuleFromFile(self._inner, modstr, cpath))
}

func (self *VM) RegisterWasmBuffer(modname string, buf []byte) error {
	modstr := toWasmEdgeStringWrap(modname)
	return newError(C.WasmEdge_VMRegisterModuleFromBuffer(self._inner, modstr, (*C.uint8_t)(unsafe.Pointer(&buf)), C.uint32_t(len(buf))))
}

func (self *VM) RegisterImport(imp *ImportObject) error {
	return newError(C.WasmEdge_VMRegisterModuleFromImport(self._inner, imp._inner))
}

func (self *VM) RegisterAST(modname string, ast *AST) error {
	modstr := toWasmEdgeStringWrap(modname)
	return newError(C.WasmEdge_VMRegisterModuleFromASTModule(self._inner, modstr, ast._inner))
}

func (self *VM) runWasm(funcname string, params ...interface{}) ([]interface{}, error) {
	res := C.WasmEdge_VMValidate(self._inner)
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	res = C.WasmEdge_VMInstantiate(self._inner)
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return self.Execute(funcname, params...)
}

func (self *VM) RunWasmFile(path string, funcname string, params ...interface{}) ([]interface{}, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	res := C.WasmEdge_VMLoadWasmFromFile(self._inner, cpath)
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return self.runWasm(funcname, params...)
}

func (self *VM) RunWasmFileWithDataAndWASI(path string, funcname string, data []byte,
	args []string, envp []string, dirs []string, preopens []string, params ...interface{}) ([]interface{}, error) {
	/// Create a temp file
	tmpf, _ := ioutil.TempFile("", "tmp.*.bin")
	defer os.Remove(tmpf.Name())
	tmpf.Write(data)
	tmpf.Close()

	/// Init WASI (test)
	var wasi = self.GetImportObject(WASI)
	if wasi != nil {
		wasi.InitWasi(
			args, /// The args
			append(envp, "WasmEdge_DATA_TO_CALLEE="+tmpf.Name()), /// The envs
			append(dirs, "/tmp:/tmp"),                            /// The mapping directories
			preopens,                                             /// The preopens will be empty
		)
	}

	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	res := C.WasmEdge_VMLoadWasmFromFile(self._inner, cpath)
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return self.runWasm(funcname, params...)
}

func (self *VM) RunWasmBuffer(buf []byte, funcname string, params ...interface{}) ([]interface{}, error) {
	res := C.WasmEdge_VMLoadWasmFromBuffer(self._inner, (*C.uint8_t)(unsafe.Pointer(&buf)), C.uint32_t(len(buf)))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return self.runWasm(funcname, params...)
}

func (self *VM) RunWasmAST(ast *AST, funcname string, params ...interface{}) ([]interface{}, error) {
	res := C.WasmEdge_VMLoadWasmFromASTModule(self._inner, ast._inner)
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return self.runWasm(funcname, params...)
}

func (self *VM) LoadWasmFile(path string) error {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	return newError(C.WasmEdge_VMLoadWasmFromFile(self._inner, cpath))
}

func (self *VM) LoadWasmBuffer(buf []byte) error {
	return newError(C.WasmEdge_VMLoadWasmFromBuffer(self._inner, (*C.uint8_t)(unsafe.Pointer(&buf)), C.uint32_t(len(buf))))
}

func (self *VM) LoadWasmAST(ast *AST) error {
	return newError(C.WasmEdge_VMLoadWasmFromASTModule(self._inner, ast._inner))
}

func (self *VM) Validate() error {
	return newError(C.WasmEdge_VMValidate(self._inner))
}

func (self *VM) Instantiate() error {
	return newError(C.WasmEdge_VMInstantiate(self._inner))
}

func (self *VM) Execute(funcname string, params ...interface{}) ([]interface{}, error) {
	funcstr := toWasmEdgeStringWrap(funcname)
	ftype := self.GetFunctionType(funcname)
	cparams := toWasmEdgeValueSlide(params...)
	creturns := make([]C.WasmEdge_Value, len(ftype._returns))
	var ptrparams *C.WasmEdge_Value = nil
	var ptrreturns *C.WasmEdge_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.WasmEdge_Value)(unsafe.Pointer(&cparams[0]))
	}
	if len(creturns) > 0 {
		ptrreturns = (*C.WasmEdge_Value)(unsafe.Pointer(&creturns[0]))
	}
	res := C.WasmEdge_VMExecute(self._inner, funcstr, ptrparams, C.uint32_t(len(cparams)), ptrreturns, C.uint32_t(len(creturns)))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return fromWasmEdgeValueSlide(creturns, ftype._returns), nil
}

func (self *VM) ExecuteRegistered(modname string, funcname string, params ...interface{}) ([]interface{}, error) {
	modstr := toWasmEdgeStringWrap(modname)
	funcstr := toWasmEdgeStringWrap(funcname)
	ftype := self.GetFunctionTypeRegistered(modname, funcname)
	cparams := toWasmEdgeValueSlide(params...)
	creturns := make([]C.WasmEdge_Value, len(ftype._returns))
	var ptrparams *C.WasmEdge_Value = nil
	var ptrreturns *C.WasmEdge_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.WasmEdge_Value)(unsafe.Pointer(&cparams[0]))
	}
	if len(creturns) > 0 {
		ptrreturns = (*C.WasmEdge_Value)(unsafe.Pointer(&creturns[0]))
	}
	res := C.WasmEdge_VMExecuteRegistered(self._inner, modstr, funcstr, ptrparams, C.uint32_t(len(cparams)), ptrreturns, C.uint32_t(len(creturns)))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return fromWasmEdgeValueSlide(creturns, ftype._returns), nil
}

func (self *VM) GetFunctionType(funcname string) *FunctionType {
	funcstr := toWasmEdgeStringWrap(funcname)
	cftype := C.WasmEdge_VMGetFunctionType(self._inner, funcstr)
	defer C.WasmEdge_FunctionTypeDelete(cftype)
	return fromWasmEdgeFunctionType(cftype)
}

func (self *VM) GetFunctionTypeRegistered(modname string, funcname string) *FunctionType {
	modstr := toWasmEdgeStringWrap(modname)
	funcstr := toWasmEdgeStringWrap(funcname)
	cftype := C.WasmEdge_VMGetFunctionTypeRegistered(self._inner, modstr, funcstr)
	defer C.WasmEdge_FunctionTypeDelete(cftype)
	return fromWasmEdgeFunctionType(cftype)
}

func (self *VM) Cleanup() {
	C.WasmEdge_VMCleanup(self._inner)
}

func (self *VM) GetFunctionList() ([]string, []*FunctionType) {
	funclen := C.WasmEdge_VMGetFunctionListLength(self._inner)
	cfnames := make([]C.WasmEdge_String, int(funclen))
	cftypes := make([]*C.WasmEdge_FunctionTypeContext, int(funclen))
	if int(funclen) > 0 {
		C.WasmEdge_VMGetFunctionList(self._inner, &cfnames[0], &cftypes[0], funclen)
	}
	fnames := make([]string, int(funclen))
	ftypes := make([]*FunctionType, int(funclen))
	for i := 0; i < int(funclen); i++ {
		fnames[i] = fromWasmEdgeString(cfnames[i])
		C.WasmEdge_StringDelete(cfnames[i])
		ftypes[i] = fromWasmEdgeFunctionType(cftypes[i])
		C.WasmEdge_FunctionTypeDelete(cftypes[i])
	}
	return fnames, ftypes
}

func (self *VM) GetImportObject(host HostRegistration) *ImportObject {
	ptr := C.WasmEdge_VMGetImportModuleContext(self._inner, C.enum_WasmEdge_HostRegistration(host))
	if ptr != nil {
		return &ImportObject{
			_inner: ptr,
		}
	}
	return nil
}

func (self *VM) GetStore() *Store {
	return &Store{
		_inner: C.WasmEdge_VMGetStoreContext(self._inner),
	}
}

func (self *VM) GetStatistics() *Statistics {
	return &Statistics{
		_inner: C.WasmEdge_VMGetStatisticsContext(self._inner),
	}
}

func (self *VM) Delete() {
	C.WasmEdge_VMDelete(self._inner)
	self._inner = nil
}
