
CREATE TABLE users (
                       id SERIAL PRIMARY KEY,
                       name VARCHAR(100) NOT NULL,
                       email VARCHAR(100) UNIQUE NOT NULL,
                       password VARCHAR(100),
                       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE groups (
                        id SERIAL PRIMARY KEY,
                        name VARCHAR(100) NOT NULL,
                        created_by INT REFERENCES users(id) ON DELETE SET NULL,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE group_users (
                             id SERIAL PRIMARY KEY,
                             group_id INT REFERENCES groups(id) ON DELETE CASCADE,
                             user_id INT REFERENCES users(id) ON DELETE CASCADE,
                             joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                             UNIQUE (group_id, user_id)
);


CREATE TABLE expenses (
                          id SERIAL PRIMARY KEY,
                          description TEXT NOT NULL,
                          amount FLOAT NOT NULL,
                          split_type VARCHAR(20) NOT NULL,
                          expense_type VARCHAR(20) NOT NULL,
                          created_by INT NOT NULL,
                          group_id INT
);

CREATE TABLE contributors (
                              id SERIAL PRIMARY KEY,
                              expense_id INT REFERENCES expenses(id) ON DELETE CASCADE,
                              user_id INT NOT NULL,
                              contribution_amount FLOAT NOT NULL,
                              paid_amount FLOAT,
                              percentage FLOAT,
                              share FLOAT,
                              amount FLOAT
);

CREATE TABLE amounts_owed (
                              id SERIAL PRIMARY KEY,
                              expense_id INT REFERENCES expenses(id) ON DELETE CASCADE,
                              user_id INT NOT NULL,
                              owed FLOAT DEFAULT 0,
                              balance FLOAT DEFAULT 0
);