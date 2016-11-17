-- Create the tables.
CREATE TABLE users (
    name NVARCHAR(320) PRIMARY KEY, -- 320 is the maximum email length.
    salt CHAR(256),
    password CHAR(256), -- Password is salted and encrypted.
    is_manager BOOL -- True if the user is also a manager (can create projects).
);
CREATE TABLE projects (
    id INT PRIMARY KEY, -- Is this required??
    name NVARCHAR(100), -- Type??
    percentage SMALLINT CHECK (percentage >= 0 and percentage <= 100),
    description NVARCHAR(512), -- Size??
    flag BOOL
);
CREATE TABLE deliverables (
    name NVARCHAR(128), -- Type?
    pid INT,
    due DATE,
    percentage SMALLINT CHECK (percentage >= 0 and percentage <= 100),
    description NVARCHAR(512), -- Size??
    PRIMARY KEY (name, pid)
);
CREATE TABLE owns (
    name NVARCHAR(320) REFERENCES users,
    pid INT REFERENCES projects,
    PRIMARY KEY (name, pid)
);
CREATE TABLE views (
    name NVARCHAR(320) REFERENCES users,
    pid INT REFERENCES projects,
    PRIMARY KEY (name, pid)
);

-- Populate the projects table.
INSERT INTO projects VALUES (0, "Test Project 1", 30, "First test project", 1);
INSERT INTO projects VALUES (1, "Test Project 2", 80, "Second test project", 0);
INSERT INTO users VALUES ("test", "", "", "true"); -- Demo account.
INSERT INTO owns VALUES ("test", 0);
INSERT INTO views VALUES ("test", 1);
