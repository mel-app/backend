# TODO #

- Sometimes InvalidMethod/Forbidden/404 is returned when another value should
  be - eg PUT /projects will return a Forbidden instead of InvalidMethod.
- I should do some performance profiling - things seem suprisingly slow.
- I should check that the database will always be in a consistent state.
- Versioning only works for the flag.
- Some values need extra validation (eg Percentage needs to be constrained)
- You can't change a project's PID, so we should fail if you PUT that.
- We don't implement disowning a project.
- The client list should probably use DELETE and PUT as partial operations?
- Testing is poor.
- Perhaps the database should be wrapped?

