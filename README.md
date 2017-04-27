# Dependencies
go get github.com/gorilla/mux

go get github.com/docker/docker/api/types 

go get github.com/docker/docker/api/types/container

go get github.com/docker/docker/client

go get github.com/docker/go-connections/nat

go get github.com/jmoiron/sqlx

go get github.com/go-sql-driver/mysql

go get golang.org/x/crypto/acme/autocert

go get golang.org/x/net/context

### Note: The below isn't necessary right now since port binding seems to make no difference
~~To resolve vendor conflicts with go-connections/nat:
rm -rf src/github.com/docker/docker/vendor

Then (once more):
go get github.com/docker/docker/client~~

# Prerequisites
Copy config.cfg.template to a new file called config.cfg and fill in your database connection details
