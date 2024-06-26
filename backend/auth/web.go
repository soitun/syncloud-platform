package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"github.com/google/uuid"
	"github.com/syncloud/platform/config"
	"github.com/syncloud/platform/parser"
	"go.uber.org/zap"
	"os"
	"path"
	"sync"
)

type Web interface {
	InitConfig() error
}

type Variables struct {
	Domain        string
	AppUrl        string
	EncryptionKey string
	JwtSecret     string
	HmacSecret    string
	DeviceUrl     string
	AuthUrl       string
	IsActivated   bool
	OIDCClients   []config.OIDCClient
}

type Authelia struct {
	mutex          *sync.Mutex
	inputDir       string
	outDir         string
	keyFile        string
	secretFile     string
	jwksKeyFile    string
	hmacSecretFile string
	userConfig     UserConfig
	systemd        Systemd
	generator      PasswordGenerator
	logger         *zap.Logger
}

type UserConfig interface {
	GetDeviceDomain() string
	DeviceUrl() string
	Url(app string) string
	OIDCClients() ([]config.OIDCClient, error)
	AddOIDCClient(client config.OIDCClient) error
	IsActivated() bool
}

type Systemd interface {
	RestartService(service string) error
}

type PasswordGenerator interface {
	Generate() (Secret, error)
}

const (
	KeyFile    = "authelia.storage.encryption.key"
	SecretFile = "authelia.jwt.secret"
	JwksKey    = "authelia.jwks.key"
	HmacSecret = "authelia.hmac_secret.key"
)

func NewWeb(
	inputDir string,
	outDir string,
	outSecretDir string,
	userConfig UserConfig,
	systemd Systemd,
	generator PasswordGenerator,
	logger *zap.Logger,
) *Authelia {
	return &Authelia{
		mutex:          &sync.Mutex{},
		inputDir:       inputDir,
		outDir:         outDir,
		keyFile:        path.Join(outSecretDir, KeyFile),
		secretFile:     path.Join(outSecretDir, SecretFile),
		jwksKeyFile:    path.Join(outSecretDir, JwksKey),
		hmacSecretFile: path.Join(outSecretDir, HmacSecret),
		userConfig:     userConfig,
		systemd:        systemd,
		generator:      generator,
		logger:         logger,
	}
}

func (w *Authelia) RegisterOIDCClient(
	id string,
	redirectURI string,
	requirePkce bool,
	tokenEndpointAuthMethod string,
) (string, error) {
	secret, err := w.generator.Generate()
	if err != nil {
		return "", err
	}

	err = w.userConfig.AddOIDCClient(config.OIDCClient{
		ID:                      id,
		Secret:                  secret.Hash,
		RedirectURI:             redirectURI,
		RequirePkce:             requirePkce,
		TokenEndpointAuthMethod: tokenEndpointAuthMethod,
	})
	if err != nil {
		return "", err
	}
	err = w.InitConfig()
	if err != nil {
		return "", err
	}
	return secret.Password, nil
}

func (w *Authelia) InitConfig() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	activated := w.userConfig.IsActivated()
	encryptionKey, err := getOrCreateUuid(w.keyFile)
	if err != nil {
		return err
	}
	jwtSecret, err := getOrCreateUuid(w.secretFile)
	if err != nil {
		return err
	}
	hmacSecret, err := getOrCreateUuid(w.hmacSecretFile)
	if err != nil {
		return err
	}
	err = createRsaKeyFileIfMissing(w.jwksKeyFile)
	if err != nil {
		return err
	}

	clients, err := w.userConfig.OIDCClients()
	if err != nil {
		return err
	}
	variables := Variables{
		Domain:        w.userConfig.GetDeviceDomain(),
		EncryptionKey: encryptionKey,
		JwtSecret:     jwtSecret,
		HmacSecret:    hmacSecret,
		DeviceUrl:     w.userConfig.DeviceUrl(),
		AuthUrl:       w.userConfig.Url("auth"),
		IsActivated:   activated,
		OIDCClients:   clients,
	}

	err = parser.Generate(
		w.inputDir,
		w.outDir,
		variables,
	)
	if err != nil {
		return err
	}

	err = w.systemd.RestartService("platform.authelia")
	if err != nil {
		w.logger.Error("unable to restart authelia", zap.Error(err))
		return err
	}

	return nil
}

func getOrCreateUuid(file string) (string, error) {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		secret := uuid.New().String()
		err = os.WriteFile(file, []byte(secret), 0644)
		return secret, err
	}
	content, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func createRsaKeyFileIfMissing(file string) error {
	_, err := os.Stat(file)
	if err == nil || !os.IsNotExist(err) {
		return err
	}
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}
	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)

	err = os.WriteFile(file, keyPEM, 0700)
	if err != nil {
		return err
	}
	return nil
}
