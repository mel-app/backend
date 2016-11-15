-- Create the tables.
CREATE TABLE users (
    name NVARCHAR(320) PRIMARY KEY, -- 320 is the maximum email length.
    salt CHAR(256),
    password CHAR(256) -- Password is salted and encrypted.
);
CREATE TABLE projects (
    id INT PRIMARY KEY, -- Is this required??
    name NVARCHAR(100), -- Type??
    percentage SMALLINT, -- Needs constraint...
    description NVARCHAR(512), -- Size??
    flag BOOL
);
CREATE TABLE views (
    name NVARCHAR(320) REFERENCES users,
    pid INT REFERENCES projects,
    PRIMARY KEY (name, pid)
);
