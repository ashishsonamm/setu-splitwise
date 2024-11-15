
# Splitwise

DB schema for this project is present in db.sql file


## Postman Collection

[Pastebin Link](https://pastebin.com/Q0MTjQRK)

## API Reference

#### Create User

```http
  POST /api/user
```

#### User Login

```http
  POST /api/login
```

#### Create Group

```http
  POST /api/group
```

#### Add User to a Group

```http
  POST /api/group/addUser
```

#### Remove User from a Group

```http
  POST /api/group/removeUser
```

#### Add expense (personal/group)

```http
  POST /api/expense
```

#### List Group Expenses

```http
  POST /api/group/{groupId}/expenses
```

#### Dashboard - Group Balances

```http
  POST /api/group/{groupId}/balances
```

#### Dashboard - Specific user balance in a group

```http
  POST /api/group/{groupId}/balances/{userId}
```

#### Dashboard - Personal user balance

```http
  POST /api/users/{userId}/balance
```

#### Dashboard - Personal user balance

```http
  POST /api/settle/{groupId}/group/{user1Id}/{user2Id}
```

## Run Locally

Specify the postgres db url in .env file

```bash
  DATABASE_URL=
```

