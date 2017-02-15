# TODO #

Also grep for TODO and FIXME in the source code.
TODOs are things that should be fixed, while FIXMEs are things that it would
be good to fix.

## Permissions/user accounts ##

- Consider authenticating *after* creating the resource and including a flag
  for when authentication is not required.

## Multiple managers ##

- Versioning only works for the flag - make that more generic.
- Support multiple managers for each project.

## Meta ##

- I should do some performance profiling - things seem suprisingly slow.
- Test coverage is sparse.
- The API should be versioned.
- Where do I document the API?

## Other ##

- Combine the "views" and "owns" tables?
- Perhaps provide a "recursive version" marker - that would allow shortcutting
  some trees when pulling from the server if the version has not changed.
  The "last changed" date is *almost* enough, bar time zones and other such
  inconsistencies.
- I should check that the database will always be in a consistent state
  (locking and atomic operations - this is also a security issue).
- Perhaps the database should be wrapped?
- We don't do proper input validation.
- Support sending JSON deltas.

