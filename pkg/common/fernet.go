package common

import (
	"sync"

	"github.com/fernet/fernet-go"

	"github.com/vCloud-DFTBA/faythe/config"
)

type FernetString struct {
	Token     string `json:"token,omitempty"`
	Encrypted bool   `json:"encrypted" default:"false"`
}

var mtx sync.Mutex

func (fs *FernetString) Encrypt() (err error) {
	mtx.Lock()
	defer mtx.Unlock()
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
	mtx.Lock()
	defer mtx.Unlock()
	if !fs.Encrypted {
		return false
	}
	k := fernet.MustDecodeKeys(config.Get().FernetKey)

	token := fernet.VerifyAndDecrypt([]byte(fs.Token), 0, k)
	fs.Token = string(token)
	fs.Encrypted = false
	return true
}
