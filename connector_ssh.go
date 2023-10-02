package fleeting_plugin_yc

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"gitlab.com/gitlab-org/fleeting/fleeting/provider"
	"golang.org/x/crypto/ssh"
)

type PrivPub interface {
	crypto.PrivateKey
	Public() crypto.PublicKey
}

func (g *InstanceGroup) ssh(info *provider.ConnectInfo) error {
	var key PrivPub
	var err error

	if info.Key != nil {
		priv, err := ssh.ParseRawPrivateKey(info.Key)
		if err != nil {
			return fmt.Errorf("reading private key: %w", err)
		}
		var ok bool
		key, ok = priv.(PrivPub)
		if !ok {
			return fmt.Errorf("key doesn't export PublicKey()")
		}
	} else {
		key, err = rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return fmt.Errorf("generating private key: %w", err)
		}
		privDER := x509.MarshalPKCS1PrivateKey(key.(*rsa.PrivateKey))
		privPEM := pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privDER,
		}
		g.settings.Key = pem.EncodeToMemory(&privPEM)

	}

	sshPubKey, err := ssh.NewPublicKey(key.Public())
	if err != nil {
		return fmt.Errorf("generating ssh public key: %w", err)
	}
	sshKey := string(ssh.MarshalAuthorizedKey(sshPubKey))[:len(ssh.MarshalAuthorizedKey(sshPubKey))-1]
	g.SshPublicKey = sshKey

	return nil
}
