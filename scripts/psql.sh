#!/bin/bash
# Run psql inside the Docker container
# Usage: ./scripts/psql.sh -c "SELECT * FROM users;"
#        ./scripts/psql.sh        (interactive mode)
docker exec -i finhelper-postgres psql -U finhelper -d finhelper "$@"
