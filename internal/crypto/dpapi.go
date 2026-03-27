package crypto

import (
	"syscall"
	"unsafe"
)

var (
	crypt32                = syscall.NewLazyDLL("crypt32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

// DPAPIEncrypt encrypts data using Windows DPAPI
func DPAPIEncrypt(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var outBlob dataBlob
	inBlob := dataBlob{
		cbData: uint32(len(data)),
		pbData: &data[0],
	}

	ret, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if ret == 0 {
		return nil, err
	}
	defer syscall.LocalFree(syscall.Handle(unsafe.Pointer(outBlob.pbData)))

	result := unsafe.Slice(outBlob.pbData, outBlob.cbData)
	output := make([]byte, outBlob.cbData)
	copy(output, result)
	return output, nil
}

// DPAPIDecrypt decrypts data using Windows DPAPI
func DPAPIDecrypt(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var outBlob dataBlob
	inBlob := dataBlob{
		cbData: uint32(len(data)),
		pbData: &data[0],
	}

	ret, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if ret == 0 {
		return nil, err
	}
	defer syscall.LocalFree(syscall.Handle(unsafe.Pointer(outBlob.pbData)))

	result := unsafe.Slice(outBlob.pbData, outBlob.cbData)
	output := make([]byte, outBlob.cbData)
	copy(output, result)
	return output, nil
}
