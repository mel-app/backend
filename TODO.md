# TODO #

- Sometimes InvalidMethod/Forbidden/404 is returned when another value should
  be - eg PUT /projects will return a Forbidden instead of InvalidMethod.
  To fix this, the permissions code should be inverted to list "Forbidden"
  accesses instead.
- In line with the above, consider authenticating *after* creating the resource
  and including a flag for when authentication is not required.
- I should do some performance profiling - things seem suprisingly slow.
- I should check that the database will always be in a consistent state.
- Versioning only works for the flag.
- Some values need extra validation (eg Percentage needs to be constrained)
- You can't change a project's PID, so we should fail if you PUT that.
- We don't implement disowning a project.
- The client list should probably use DELETE and PUT as partial operations?
- Testing is poor.
- Perhaps the database should be wrapped?
- POSTs should return a LOCATION header and a different status code.
- Wrap the version code and make it generic; apply it to everything.
- Test coverage is sparse.
- Implement DELETE for user accounts.
- Support multiple managers for each project.
- Consider supporting "sharing" projects?
- Fix the client support so that managers only send changes to the client list.

