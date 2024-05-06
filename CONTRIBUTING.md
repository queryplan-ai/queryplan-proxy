
In order to build the proxy side that talks to the website, you need to set the following environment variables after creating a MySQL Database:

```
export QUERYPLAN_LIVE_CONNECTION_URI=root:password@tcp(localhost:3307)/queryplantst
export QUERYPLAN_DATABASE_NAME=queryplantst
export QUERYPLAN_DBMS=mysql
export QUERYPLAN_BIND_ADDRESS=0.0.0.0
export QUERYPLAN_BIND_PORT=3307
export QUERYPLAN_UPSTREAM_ADDRESS=127.0.0.1
export QUERYPLAN_UPSTREAM_PORT=3306
export QUERYPLAN_API_URL=http://localhost:3100
export QUERYPLAN_TOKEN=<Will be displayed in the QueryPlan site>

export QUERYPLAN_ENV=marccampbell
```

After these have been set, replacing "password" in the connection URI with the password you want to use for the database, and "queryplantst" with the name of the database you want to use for the proxy, you can build the proxy with the following command, as long as the QueryPlan is listening. Check out the QueryPlan CONTRIBUTING.md if not already to make sure you have the QueryPlan site running.

```
make build run
```