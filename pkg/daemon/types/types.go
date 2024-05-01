package types

type DBMS string

const (
	Postgres DBMS = "postgres"
	Mysql    DBMS = "mysql"
)

type DaemonOpts struct {
	APIURL      string
	Token       string
	Environment string

	DBMS DBMS

	LiveConnectionURI string
	DatabaseName      string

	BindAddress string
	BindPort    int

	UpstreamAddress string
	UpstreamPort    int
}
