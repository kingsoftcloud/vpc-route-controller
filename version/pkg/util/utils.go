package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"

	"k8s.io/klog"
)

const (
	// Bits is the number of bits in a UUID
	Bits = 128

	// Size is the number of bytes in a UUID
	Size = Bits / 8

	format = "%08x-%04x-%04x-%04x-%012x"
)

type UUID [Size]byte

func (u UUID) String() string {
	return fmt.Sprintf(format, u[:4], u[4:6], u[6:8], u[8:10], u[10:])
}

func Generate() (u UUID) {
	const (
		// ensures we backoff for less than 450ms total. Use the following to
		// select new value, in units of 10ms:
		// 	n*(n+1)/2 = d -> n^2 + n - 2d -> n = (sqrt(8d + 1) - 1)/2
		maxretries = 9
		backoff    = time.Millisecond * 10
	)

	var (
		totalBackoff time.Duration
		count        int
		retries      int
	)

	for {
		// This should never block but the read may fail. Because of this,
		// we just try to read the random number generator until we get
		// something. This is a very rare condition but may happen.
		b := time.Duration(retries) * backoff
		time.Sleep(b)
		totalBackoff += b

		n, err := io.ReadFull(rand.Reader, u[count:])
		if err != nil {
			if retryOnError(err) && retries < maxretries {
				count += n
				retries++
				klog.Errorf("error generating version 4 uuid, retrying: %v", err)
				continue
			}

			// Any other errors represent a system problem. What did someone
			// do to /dev/urandom?
			panic(fmt.Errorf("error reading random number generator, retried for %v: %v", totalBackoff.String(), err))
		}

		break
	}

	u[6] = (u[6] & 0x0f) | 0x40 // set version byte
	u[8] = (u[8] & 0x3f) | 0x80 // set high order byte 0b10{8,9,a,b}

	return u
}

func retryOnError(err error) bool {
	switch err := err.(type) {
	case *os.PathError:
		return retryOnError(err.Err) // unpack the target error
	case syscall.Errno:
		if err == syscall.EPERM {
			// EPERM represents an entropy pool exhaustion, a condition under
			// which we backoff and retry.
			return true
		}
	}

	return false
}

func AesDecrypt(cryted string, key string) string {
	// 转成字节数组
	crytedByte, _ := base64.StdEncoding.DecodeString(cryted) //base解码
	k := []byte(key)
	// 分组秘钥
	block, _ := aes.NewCipher(k)
	// 获取秘钥块的长度
	blockSize := block.BlockSize()
	// 加密模式
	blockMode := cipher.NewCBCDecrypter(block, k[:blockSize])
	// 创建数组
	orig := make([]byte, len(crytedByte))
	// 解密
	blockMode.CryptBlocks(orig, crytedByte)
	// 去补全码
	orig = PKCS7UnPadding(orig)
	return string(orig)
}

//去码
func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
