# Database #

### users ###

- NVARCHAR(320) _name_
- CHAR(512)? password

### projects ###

Projects must have *at least one* owner.

- INT? _id_ (needs this to be unique)
- name?     (can perhaps replace the id?)
- SMALLINT percentage (should also be constrained)
- NVARCHAR(512)? description?
- BOOL flag

### deliverables ###

This is a weak entity.

- NVARCHAR(128) _name_
- INT? _pid_ (this references project)
- DATE due
- SMALLINT percentage (should also be constrained)
- NVARCHAR(512)? description?
- DATE update\_date?
- TIME update\_time??

### owns ###

All projects must have *at least one* owner.

- NVARCHAR(320) _name_ (references users)
- INT? _pid_ (references project)

### views ###

- NVARCHAR(320) _name_ (references users)
- INT? _pid_ (references project)

