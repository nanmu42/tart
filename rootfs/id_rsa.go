package rootfs

import (
	_ "embed"
	"fmt"

	"golang.org/x/crypto/ssh"
)

//go:embed id_rsa
var prikey []byte

var SSHSigner ssh.Signer

func init() {
	var err error
	SSHSigner, err = ssh.ParsePrivateKey(prikey)
	if err != nil {
		err = fmt.Errorf("ssh.ParsePrivateKey: %w", err)
		panic(err)
	}
}
