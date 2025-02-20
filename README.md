# Genji

[![Build Status](https://travis-ci.org/asdine/genji.svg)](https://travis-ci.org/asdine/genji)
[![GoDoc](https://godoc.org/github.com/asdine/genji?status.svg)](https://godoc.org/github.com/asdine/genji)
[![Slack channel](https://img.shields.io/badge/slack-join%20chat-green.svg)](https://gophers.slack.com/messages/CKPCYQFE0)

Genji is an embedded SQL database build on top of key-value stores. It supports various engines that write data on-disk, like [BoltDB](https://github.com/etcd-io/bbolt) and [Badger](https://github.com/dgraph-io/badger), or in memory.

Genji tables are schemaless and can be manipulated using SQL queries. Genji is also compatible with the `database/sql` package.

## Installation

Install the Genji library and command line tool

``` bash
go get -u github.com/asdine/genji/...
```

## Usage

There are two ways of using Genji, either by using Genji's API or by using the [`database/sql`](https://golang.org/pkg/database/sql/) package.

### Using Genji's API

```go
// Instantiate an engine, here we'll store everything in memory
ng := memory.NewEngine()

// Create a database instance
db, err := genji.New(ng)
// Don't forget to close the database when you're done
defer db.Close()

// Create a table. Genji tables are schemaless, you don't need to specify a schema.
err = db.Exec("CREATE TABLE user")

// Create an index.
err = db.Exec("CREATE INDEX idx_user_Name ON test (Name)")

// Insert some data
err = db.Exec("INSERT INTO user (ID, Name, Age) VALUES (?, ?, ?)", 10, "Foo1", 15)
err = db.Exec("INSERT INTO user (ID, Name, Age) VALUES (?, ?, ?)", 11, "Foo2", 20)

// Use a transaction
tx, err := db.Begin(true)
defer tx.Rollback()
err = tx.Exec("INSERT INTO user (ID, Name, Age) VALUES (?, ?, ?)", 12, "Foo3", 25)
...
err = tx.Commit()

// Query some records
res, err := db.Query("SELECT * FROM user WHERE Age > ?", 18)
// always close the result when you're done with it
defer res.Close()

// Iterate over the results
err = res.Iterate(func(r record.Record) error {
    var id int
    var name string
    var age int32

    err = recordutil.Scan(r, &id, &name, &age)
    if err != nil {
        return err
    }

    fmt.Println(id, name, age)
    return nil
})

// Count results
count, err := res.Count()

// Get first record from the results
r, err := res.First()
var id int
var name string
var age int32
err = recordutil.Scan(r, &id, &name, &age)

// Apply some transformations
err = res.
    // Filter all even ids
    Filter(func(r record.Record) (bool, error) {
        f, err := r.GetField("ID")
        ...
        id, err := f.DecodeToInt()
        ...
        return id % 2 == 0, nil
    }).
    // Enrich the records with a new field
    Map(func(r record.Record) (record.Record, error) {
        var fb record.FieldBuffer

        err := fb.ScanRecord(r)
        ...
        fb.Add(record.NewStringField("Group", "admin"))
        return &fb, nil
    }).
    // Iterate on them
    Iterate(func(r record.Record) error {
        ...
    })
```

### Using database/sql

```go
// Instantiate an engine, here we'll store everything in memory
ng := memory.NewEngine()

// Create a sql/database DB instance
db, err := genji.Open(ng)
defer db.Close()

// Then use db as usual
res, err := db.ExecContext(...)
res, err := db.Query(...)
res, err := db.QueryRow(...)
```

## Code generation

Genji also supports structs as long as they implement the `record.Record` interface for writes and the `record.Scanner` interface for reads.
To simplify implementing these interfaces, Genji provides a command line tool that can generate methods for you.

Declare a structure. Note that, even though struct tags are defined, Genji **doesn't use reflection** for these structures.

``` go
// user.go

type User struct {
    ID int64    `genji:"pk"`
    Name string
    Age int
}
```

Generate code to make that structure compatible with Genji.

``` bash
genji -f user.go -s User
```

This command generates a file that adds methods to the `User` type.

``` go
// user.genji.go

// The User type gets new methods that implement some Genji interfaces.
func (u *User) GetField(name string) (record.Field, error) {}
func (u *User) Iterate(fn func(record.Field) error) error {}
func (u *User) ScanRecord(rec record.Record) error {}
func (u *User) Scan(src interface{}) error
func (u *User) PrimaryKey() ([]byte, error) {}
```

### Example

``` go
// Let's create a user
u1 := User{
    ID: 20,
    Name: "foo",
    Age: 40,
}

// Let's create a few other ones
u2 := u1
u2.ID = 21
u3 := u1
u3.ID = 22

// It is possible to let Genji deal with analyzing the structure
// when inserting a record, using the RECORDS clause
err := db.Exec(`INSERT INTO user RECORDS ?, ?, ?`, &u1, &u2, &u3)
// Note that it is also possible to write records by hand
err := db.Exec(`INSERT INTO user RECORDS ?, (ID: 21, Name: "foo", Age: 40), ?`, &u1, &u3)

// Let's select a few users
var users []User

res, err := db.Query("SELECT * FROM user")
defer res.Close()

err = res.Iterate(func(r record.Record) error {
    var u User
    // Use the generated ScanRecord method this time
    err := u.ScanRecord(r)
    if err != nil {
        return err
    }

    users = append(users, u)
    return nil
})
```

## Engines

Genji currently supports storing data in [BoltDB](https://github.com/etcd-io/bbolt), [Badger](https://github.com/dgraph-io/badger) and in-memory.

### Use the BoltDB engine

``` go
import (
    "log"

    "github.com/asdine/genji"
    "github.com/asdine/genji/engine/bolt"
)

func main() {
    // Create a bolt engine
    ng, err := bolt.NewEngine("genji.db", 0600, nil)
    if err != nil {
        log.Fatal(err)
    }

    // Pass it to genji
    db := genji.New(ng)
    defer db.Close()
}
```

### Use the Badger engine

``` go
import (
    "log"

    "github.com/asdine/genji"
    "github.com/asdine/genji/engine/badger"
    bdg "github.com/dgraph-io/badger"
)

func main() {
    // Create a badger engine
    ng, err := badger.NewEngine(bdg.DefaultOptions("genji")))
    if err != nil {
        log.Fatal(err)
    }

    // Pass it to genji
    db, err := genji.New(ng)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
}
```

### Use the memory engine

``` go
import (
    "log"

    "github.com/asdine/genji"
    "github.com/asdine/genji/engine/memory"
)

func main() {
    // Create a memory engine
    ng := memory.NewEngine()

    // Pass it to genji
    db, err := genji.New(ng)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
}
```

## Tags

Genji scans the struct tags at compile time, not at runtime, and it uses this information to generate code.

Here is the description of the only supported tag:

* `pk` : Indicates that this field is the primary key. The primary key can be of any type. If this tag is not provided, Genji uses its own internal autoincremented id
