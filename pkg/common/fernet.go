package common

import (
	"github.com/fernet/fernet-go"
	"sync"

	"github.com/vCloud-DFTBA/faythe/config"
)

type FernetString struct {
	Token     string `json:"token"`
	Encrypted bool   `json:"encrypted" default:"false"`
	mtx       sync.Mutex
}

func (fs *FernetString) Encrypt() (err error) {
	fs.mtx.Lock()
	defer fs.mtx.Unlock()
	if fs.Encrypted {
		return nil
	}
	k := fernet.MustDecodeKeys(config.Get().FernetKey)
	token, err := fernet.EncryptAndSign([]byte(fs.Token), k[0])
	if err != nil {
		return err
	}
	fs.Encrypted = true
	fs.Token = string(token)
	return nil
}

func (fs *FernetString) Decrypt() bool {
	fs.mtx.Lock()
	defer fs.mtx.Unlock()
	if !fs.Encrypted {
		return false
	}
	k := fernet.MustDecodeKeys(config.Get().FernetKey)

	token := fernet.VerifyAndDecrypt([]byte(fs.Token), 0, k)
	fs.Token = string(token)
	fs.Encrypted = false
	return true
}
