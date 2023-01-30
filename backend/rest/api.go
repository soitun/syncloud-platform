package rest

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/syncloud/platform/info"
	"go.uber.org/zap"
	"net"
	"net/http"
)

type DeviceUserConfig interface {
	GetDeviceDomain() string
	GetDkimKey() *string
	SetDkimKey(key *string)
	GetUserEmail() *string
}

type Storage interface {
	InitAppStorageOwner(app, owner string) (string, error)
	GetAppStorageDir(app string) string
}

type Systemd interface {
	RestartService(service string) error
}

type Api struct {
	device     *info.Device
	userConfig DeviceUserConfig
	storage    Storage
	systemd    Systemd
	logger     *zap.Logger
}

func NewApi(device *info.Device, userConfig DeviceUserConfig, storage Storage, systemd Systemd, logger *zap.Logger) *Api {
	return &Api{
		device:     device,
		userConfig: userConfig,
		storage:    storage,
		systemd:    systemd,
		logger:     logger,
	}
}

func (a *Api) Start(network string, address string) {
	listener, err := net.Listen(network, address)
	if err != nil {
		panic(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/app/install_path", Handle(a.AppInstallPath)).Methods("GET")
	r.HandleFunc("/app/data_path", Handle(a.AppDataPath)).Methods("GET")
	r.HandleFunc("/app/url", Handle(a.AppUrl)).Methods("GET")
	r.HandleFunc("/app/domain_name", Handle(a.AppDomainName)).Methods("GET")
	r.HandleFunc("/app/device_domain_name", Handle(a.AppDeviceDomainName)).Methods("GET")
	r.HandleFunc("/app/init_storage", Handle(a.AppInitStorage)).Methods("POST")
	r.HandleFunc("/config/get_dkim_key", Handle(a.ConfigGetDkimKey)).Methods("GET")
	r.HandleFunc("/config/set_dkim_key", Handle(a.ConfigSetDkimKey)).Methods("POST")
	r.HandleFunc("/service/restart", Handle(a.ServiceRestart)).Methods("POST")
	r.HandleFunc("/app/storage_dir", Handle(a.AppStorageDir)).Methods("GET")
	r.HandleFunc("/user/email", Handle(a.UserEmail)).Methods("GET")
	r.NotFoundHandler = http.HandlerFunc(notFoundHandler)

	r.Use(middleware)

	fmt.Println("Started api")
	_ = http.Serve(listener, r)

}

func (a *Api) AppInstallPath(req *http.Request) (interface{}, error) {
	keys, ok := req.URL.Query()["name"]
	if !ok {
		return nil, fmt.Errorf("no name")
	}
	return fmt.Sprintf("/snap/%s/current", keys[0]), nil
}

func (a *Api) AppDataPath(req *http.Request) (interface{}, error) {
	keys, ok := req.URL.Query()["name"]
	if !ok {
		return nil, fmt.Errorf("no name")
	}
	return fmt.Sprintf("/var/snap/%s/common", keys[0]), nil
}

func (a *Api) AppUrl(req *http.Request) (interface{}, error) {
	keys, ok := req.URL.Query()["name"]
	if !ok {
		return nil, fmt.Errorf("no name")
	}
	return a.device.Url(keys[0]), nil
}

func (a *Api) AppDomainName(req *http.Request) (interface{}, error) {
	keys, ok := req.URL.Query()["name"]
	if !ok {
		return nil, fmt.Errorf("no name")
	}
	return a.device.AppDomain(keys[0]), nil
}

func (a *Api) AppDeviceDomainName(_ *http.Request) (interface{}, error) {
	return a.userConfig.GetDeviceDomain(), nil
}

func (a *Api) AppInitStorage(req *http.Request) (interface{}, error) {
	err := req.ParseForm()
	if err != nil {
		return nil, err
	}
	return a.storage.InitAppStorageOwner(req.FormValue("app_name"), req.FormValue("user_name"))
}

func (a *Api) ConfigGetDkimKey(_ *http.Request) (interface{}, error) {
	return a.userConfig.GetDkimKey(), nil
}

func (a *Api) ConfigSetDkimKey(req *http.Request) (interface{}, error) {
	err := req.ParseForm()
	if err != nil {
		return nil, err
	}
	key := req.FormValue("dkim_key")
	a.userConfig.SetDkimKey(&key)
	return "OK", nil
}

func (a *Api) ServiceRestart(req *http.Request) (interface{}, error) {
	err := req.ParseForm()
	if err != nil {
		return nil, err
	}
	err = a.systemd.RestartService(req.FormValue("name"))
	return "OK", err
}

func (a *Api) AppStorageDir(req *http.Request) (interface{}, error) {
	keys, ok := req.URL.Query()["name"]
	if !ok {
		return nil, fmt.Errorf("no name")
	}
	return a.storage.GetAppStorageDir(keys[0]), nil
}

func (a *Api) UserEmail(_ *http.Request) (interface{}, error) {
	return a.userConfig.GetUserEmail(), nil
}
