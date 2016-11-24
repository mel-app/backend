# Server API #

To update, a client pushes any changes to the server, then pulls everything.
Merges are done on the server to simplify client-side development.

The API needs to be versioned.
Credentials are sent with each request to minimise server-side state.
I should try to minimize network traffic - how many bytes is a single https
request?

The server needs timestamps on all properties to ensure that everything can be
syncronised.


## Structure ##

TODO: This may be better replaced with just sending deltas...

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

Use PUT to update existing objects, POST to request creating a new object, and
GET to get the current state.


## Syncronising ##

For projects with more than one project manager, I'll need to be careful to
ensure that changes can be syncronised.
For flags, I'll need to deal with both clients and managers changing the state.

====================>
  ^     ^    ^    ^
  1     2    3   Now

Consider the above situation. Assume that 1, 2, 3 are different devices syncing
to the server, and that the server has the "canonical" state.

For simplicity, only consider the case where the flag is being set/unset, and
additionally assume that device #2 is a PM, while #1, #3 are client devices.

- If #1 syncs and sets the flag, but the either #2 or #3 unset it, ignore.
- If #1 syncs and sets the flag, and it has not been unset since the last
  sync, set the flag.
- If #2 syncs and unsets the flag, but #3 has set the flag, set the flag?
- If #2 syncs and unsets the flag, and it has not been set since the last
  sync, unset the flag.
- If #3 syncs and sets the flag, always set the flag.

All attributes needs to have a "last changed revision", and a per-device
"last sync revision".

From the server perspective, if a device is syncronising and has a change that
would change the flag state, only apply the change if the revision number is
>= than flag revision number.
The server then needs to set the revision number to that number plus 1.

