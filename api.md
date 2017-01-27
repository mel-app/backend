# Server API #

To update, a client pushes any changes to the server, then pulls everything.
Merges are done on the server to simplify client-side development.

There are distinct "items" on the server side, and "lists".
"lists" nodes are represented by a line-by-line series of JSON tokens,
containing identifiers for the "items" directly under that node.
"items" nodes are represented by a single JSON blob.

## Authentication ##

The server recognizes basic http authenication; as such, it is vital to ensure
that all communication with the server is encrypted (https).

The /login node lets the client interact with the login information.
To check authentication details, try a GET to /login; if that succeeds, the
login details must be valid. /login also allows login deletion (DELETE),
updating the password (PUT), and creation (POST).

## Structure ##

pID: project ID
dID: deliverable ID

Note that I should url encode the ID's if I am using names, and unencode them
otherwise.

- login: login creation and handling
- projects: list of projects accessible to the user, can-create permissions
- projects/pID: project properties (percentage, description)
- projects/pID/flag: current flag state
- projects/pID/clients: list of project clients
- projects/pID/deliverables: list of project deliverables
- projects/pID/deliverables/dID: deliverable state

For items, use GET to retrieve, DELETE to remove, and PUT to update.
For lists, use GET to retrieve, POST to request creating a new object.

## Syncronising ##

Some elements on the server are "pushed" to from more than one client.
These elements need to be synchronised; this is accomplished through
server-side versioning.
The system "trusts" that the client responds with the correct "version number",
however since the client does not need to care about the version number that
should be reasonably safe.

====================>
  ^     ^    ^    ^
  1     2    3   Now

Consider the above situation. Assume that 1, 2, 3 are different devices syncing
to the server, and that the server has the "canonical" state.

For simplicity, only consider the case where a boolean flag is being set/unset.

- If #1 syncs and sets the flag, but the either #2 or #3 unset it, ignore.
- If #1 syncs and sets the flag, and it has not been unset since the last
  sync, set the flag.
- If #2 syncs and unsets the flag, but #3 has set the flag, set the flag?
- If #2 syncs and unsets the flag, and it has not been set since the last
  sync, unset the flag.
- If #3 syncs and sets the flag, always set the flag.

From the server perspective, if a device is syncronising and has a change that
would change the flag state, only apply the change if the revision number from
the device is >= than server's revision number.
The server then needs to set the revision number to that number plus 1.

