package types

type DBMS string

const (
	Postgres DBMS = "postgres"
	Mysql    DBMS = "mysql"
)

type DaemonOpts struct {
	DBMS DBMS

	BindAddress string
	BindPort    int

	UpstreamAddress string
	UpstreamPort    int
}
