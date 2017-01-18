# tests/ #

This directory contains end-to-end tests of the backend code.
As such, it requires a running postgresql server. To set up a postgresql
server, install the appropriate package, and then run

    $ initdb <path/to/db>

to complete the setup.

Edit the postgresql.conf file in the db root, and change
<code>unix_socket_directories</code> to somewhere your user can access
(eg <code>/run/user/<user id>/</code>).
Finally, run

    $ pg_ctl -D <path/to/db> start
    $ create_db -h <path/to/new/unix/socket/dir> backend-test

to start the server and create a new database for testing with.
