package mysql

import "github.com/queryplan-ai/queryplan-proxy/test/integration/pkg/upstream"

func Execute() ([]*upstream.UpsreamProcess, error) {
	upstreamProcesses := []*upstream.UpsreamProcess{}

	standardUpstream, err := executeStandardQueries()
	if err != nil {
		return nil, err
	}
	upstreamProcesses = append(upstreamProcesses, standardUpstream)

	return upstreamProcesses, nil
}

func executeStandardQueries() (*upstream.UpsreamProcess, error) {
	u, err := upstream.StartMysql(false, false)
	if err != nil {
		return nil, err
	}

	return u, nil
}
