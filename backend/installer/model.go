package installer

type AppInstaller interface {
	Upgrade() error
}
