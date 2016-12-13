# TODO #

Also grep for TODO and FIXME in the source code.
TODOs are things that should be fixed, while FIXMEs are things that it would
be good to fix.

## Permissions/user accounts ##

- Sometimes InvalidMethod/Forbidden/404 is returned when another value should
  be - eg PUT /projects will return a Forbidden instead of InvalidMethod.
  To fix this, the permissions code should be inverted to list "Forbidden"
  accesses instead.
- In line with the above, consider authenticating *after* creating the resource
  and including a flag for when authentication is not required.
- Implement DELETE for user accounts.

## Multiple managers ##

- Versioning only works for the flag.
- Wrap the version code and make it generic; apply it to everything.
- Support multiple managers for each project.
- Consider supporting "sharing" projects?

## Meta ##

- I should do some performance profiling - things seem suprisingly slow.
- Test coverage is sparse.
- The API should be versioned.
- Where do I document the API?

## Other ##

- I should check that the database will always be in a consistent state.
- Perhaps the database should be wrapped?
- We don't tell the user if they are a manager or not.

